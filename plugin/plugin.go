// Copyright (c) 2023, the Drone Plugins project authors.
// Please see the AUTHORS file for details. All rights reserved.
// Use of this source code is governed by an Apache 2.0 license that can be
// found in the LICENSE file.

package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/appleboy/drone-git-push/repo"
	"github.com/drone/drone-go/drone"
	"github.com/sirupsen/logrus"
)

// Args provides plugin execution arguments.
type (
	Args struct {
		Pipeline

		// Level defines the plugin log level.
		Level string `envconfig:"PLUGIN_LOG_LEVEL"`

		// Skip verification of certificates
		SkipVerify bool `envconfig:"PLUGIN_SKIP_VERIFY"`

		// Lint plugin
		Lint bool `envconfig:"PLUGIN_LINT" default:"true"`

		// Plugin specific
		Key             string `envconfig:"PLUGIN_SSH_KEY"`
		PagesDirectory  string `envconfig:"PLUGIN_PAGES_DIRECTORY"`
		TargetDirectory string `envconfig:"PLUGIN_TARGET_DIRECTORY"`

		PagesRepo struct {
			Remote   string `envconfig:"PLUGIN_REMOTE_URL"`
			Branch   string `envconfig:"PLUGIN_TARGET_BRANCH"`
			Name     string `envconfig:"PLUGIN_UPSTREAM_NAME"`
			Checkout string
		}

		Rsync struct {
			ExcludeCname bool `envconfig:"PLUGIN_EXCLUDE_CNAME"`
			Delete       bool `envconfig:"PLUGIN_DELETE"`
			CopyContents bool `envconfig:"PLUGIN_COPY_CONTENTS"`
			Source       string
			Destination  string
		}

		PagesCommit struct {
			Message   string `envconfig:"PLUGIN_MESSAGE"`
			ForcePush bool   `envconfig:"PLUGIN_FORCE_PUSH"`
			Author    struct {
				Name  string `envconfig:"PLUGIN_USER_NAME"`
				Email string `envconfig:"PLUGIN_USER_EMAIL"`
			}
		}

		Netrc struct {
			Machine  string `envconfig:"PLUGIN_NETRC_MACHINE"`
			Login    string `envconfig:"PLUGIN_USERNAME"`
			Password string `envconfig:"PLUGIN_PASSWORD"`
		}
	}
)

var errConfiguration = errors.New("configuration error")

// Exec executes the plugin.
func Exec(ctx context.Context, args *Args) error {
	linter := ""

	if args.Lint {
		issues, warnings := lintArgs(args)
		linter = fmt.Sprintf("lint: %d issue(s) found\n%s", issues, warnings)
		logrus.Infof("%s\n", linter)
	}

	err := verifyArgs(args)
	if err != nil {
		return fmt.Errorf("error in the configuration: %w", err)
	}

	// Prepare git config
	err = prepare(args)
	if err != nil {
		return fmt.Errorf("error configuring git: %w", err)
	}

	// Run the plugin
	err = process(args)
	if err != nil {
		return fmt.Errorf("error during processing: %w", err)
	}

	// Get pages link
	pages, err := pagesURL(args)
	if err != nil {
		logrus.Warningf("could not determine location for site, skipping card\n")

		return nil //nolint:nilerr
	}

	logrus.Infof("publishing at: %s\n", pages)

	// Create the card data
	cardData := struct {
		URL    string `json:"url"`
		Linter string `json:"linter"`
	}{
		URL:    pages.String(),
		Linter: linter,
	}

	data, _ := json.Marshal(cardData)
	card := drone.CardInput{
		Schema: "https://drone-plugins.github.io/drone-gh-pages/card.json",
		Data:   data,
	}
	writeCard(args.Card.Path, &card)

	return nil
}

func lintArgs(args *Args) (issues int, warnings string) {
	issues = 0

	var warningsBuilder strings.Builder

	if args.PagesRepo.Name != "" {
		warningsBuilder.WriteString("remove upstream_name from config it is deprecated\n")
		issues++
	}

	if args.Netrc.Machine != "" {
		warningsBuilder.WriteString("remove netrc_machine from config is it deprectated\n")
		issues++
	}

	if _, present := os.LookupEnv("PLUGIN_TEMPORARY_BASE"); present {
		warningsBuilder.WriteString("remove temporary_base from config it is deprecated\n")
		issues++
	}

	if args.PagesRepo.Remote == os.Getenv("DRONE_REPO_LINK") {
		warningsBuilder.WriteString("remove remote_url as its value is redundant\n")
		issues++
	}

	if args.Key != "" && args.Netrc.Password != "" {
		warningsBuilder.WriteString("both key and password are being set, choose one auth method\n")
		issues++
	}

	if strings.HasSuffix(args.PagesDirectory, "/") {
		warningsBuilder.WriteString("remove trailing slash from pages_directory and set copy_contents to `true` to rsync the contents of the directory")
		issues++
	}

	if strings.HasSuffix(args.TargetDirectory, "/") {
		warningsBuilder.WriteString("remove trailing slash from target_directory and set copy_contents to `true` to rsync the contents of the directory")
		issues++
	}

	return issues, warningsBuilder.String()
}

func verifyArgs(args *Args) error {
	if args.Key == "" && args.Netrc.Password == "" {
		return fmt.Errorf("no authentication method specified: %w", errConfiguration)
	}

	if args.PagesDirectory == "" {
		args.PagesDirectory = "docs"
	}

	if args.TargetDirectory == "" {
		args.TargetDirectory = "."
	}

	// PagesRepo
	if args.PagesRepo.Remote == "" {
		args.PagesRepo.Remote = os.Getenv("DRONE_REMOTE_URL")
		if args.PagesRepo.Remote == "" {
			return fmt.Errorf("gh-pages remote not specified: %w", errConfiguration)
		}
	}

	remoteURI, err := url.Parse(args.PagesRepo.Remote)
	if err != nil {
		return fmt.Errorf("invalid clone url: %w", err)
	}

	if args.PagesRepo.Name == "" {
		args.PagesRepo.Name = "origin"
	}

	if args.PagesRepo.Branch == "" {
		args.PagesRepo.Branch = "gh-pages"
	}

	tmp, err := os.MkdirTemp("", "drone-gh-pages")
	if err != nil {
		return fmt.Errorf("could not create temporary directory: %w", err)
	}

	args.PagesRepo.Checkout = tmp

	// PagesCommit
	if args.PagesCommit.Author.Name == "" {
		args.PagesCommit.Author.Name = args.Commit.Author.Name
		if args.PagesCommit.Author.Name == "" {
			return fmt.Errorf("author name not specified: %w", errConfiguration)
		}
	}

	if args.PagesCommit.Author.Email == "" {
		args.PagesCommit.Author.Email = args.Commit.Author.Email
		if args.PagesCommit.Author.Email == "" {
			return fmt.Errorf("author email not specified: %w", errConfiguration)
		}
	}

	if args.PagesCommit.Message == "" {
		args.PagesCommit.Message = args.Commit.Message
		if args.PagesCommit.Message == "" {
			return fmt.Errorf("commit message not specified: %w", errConfiguration)
		}
	} else {
		args.PagesCommit.Message, err = contents(args.PagesCommit.Message)
		if err != nil {
			return fmt.Errorf("commit message not specified: %w", errConfiguration)
		}
	}

	// Rsync
	if !filepath.IsAbs(args.PagesDirectory) {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("could not get working directory: %w", err)
		}

		args.Rsync.Source = filepath.Join(wd, args.PagesDirectory)

		_, err = os.Stat(args.Rsync.Source)
		if err != nil {
			return fmt.Errorf("could not get pages directory: %w", err)
		}
	} else {
		args.Rsync.Source = args.PagesDirectory
	}

	if args.Rsync.CopyContents {
		args.Rsync.Source += "/"
	}

	if filepath.IsAbs(args.TargetDirectory) {
		return fmt.Errorf("target_directory needs to be relative: %w", errConfiguration)
	}

	args.Rsync.Destination = filepath.Join(args.PagesRepo.Checkout, args.TargetDirectory)

	// Netrc
	args.Netrc.Machine = remoteURI.Hostname()

	return nil
}

func prepare(args *Args) error {
	if args.Netrc.Login != "" && args.Netrc.Password != "" {
		if err := repo.WriteNetrc(args.Netrc.Machine, args.Netrc.Login, args.Netrc.Password); err != nil {
			return fmt.Errorf("failed to write netrc: %w", err)
		}

		logrus.Infof("using netrc file for authentication: machine %s login %s\n", args.Netrc.Machine, args.Netrc.Login)
	}

	if args.Key != "" {
		if err := repo.WriteKey(args.Key); err != nil {
			return fmt.Errorf("failed to write ssh key: %w", err)
		}

		logrus.Infof("using ssh key for authentication\n")
	}

	if err := repo.GlobalUser(args.PagesCommit.Author.Email).Run(); err != nil {
		return fmt.Errorf("failed to set email: %w", err)
	}

	if err := repo.GlobalName(args.PagesCommit.Author.Name).Run(); err != nil {
		return fmt.Errorf("failed to set author: %w", err)
	}

	logrus.Infof("committing as: %s <%s>\n", args.PagesCommit.Author.Name, args.PagesCommit.Author.Email)

	if args.SkipVerify {
		if err := repo.SkipVerify().Run(); err != nil {
			return fmt.Errorf("failed to disable ssl verification: %w", err)
		}

		logrus.Warningf("ssl verification is turned off")
	}

	return nil
}

func process(args *Args) error {
	defer os.RemoveAll(args.PagesRepo.Checkout)

	if err := cloneTarget(args); err != nil {
		return fmt.Errorf("failed to clone target: %w", err)
	}

	if err := rsyncPages(args); err != nil {
		return fmt.Errorf("failed to sync pages: %w", err)
	}

	if dirtyRepo(args) {
		if err := stageChanges(args); err != nil {
			return fmt.Errorf("failed to stage changes: %w", err)
		}

		if err := commitChanges(args); err != nil {
			return fmt.Errorf("failed to commit changes: %w", err)
		}

		if err := pushChanges(args); err != nil {
			return fmt.Errorf("failed to push changes: %w", err)
		}
	} else {
		logrus.Infof("no changes detected on branch\n")
	}

	return nil
}

func cloneTarget(args *Args) error {
	clone := []string{
		"clone",
		"--branch",
		args.PagesRepo.Branch,
		"--origin",
		args.PagesRepo.Name,
		"--single-branch",
		args.PagesRepo.Remote,
		args.PagesRepo.Checkout,
	}

	cmd := exec.Command(
		"git",
		clone...,
	)

	return runCommand(cmd)
}

func rsyncPages(args *Args) error {
	rysnc := []string{
		"-r",
		"--exclude",
		".git",
	}

	if args.Rsync.ExcludeCname {
		rysnc = append(
			rysnc,
			"--exclude",
			"CNAME",
		)
	}

	if args.Rsync.Delete {
		rysnc = append(
			rysnc,
			"--delete",
		)
	}

	rysnc = append(
		rysnc,
		args.Rsync.Source,
		args.Rsync.Destination,
	)

	cmd := exec.Command(
		"rsync",
		rysnc...,
	)

	return runCommand(cmd)
}

func stageChanges(args *Args) error {
	cmd := exec.Command(
		"git",
		"add",
		".",
	)
	cmd.Dir = args.PagesRepo.Checkout

	return runCommand(cmd)
}

func commitChanges(args *Args) error {
	commit := []string{
		"commit",
		"-m",
		args.PagesCommit.Message,
	}

	cmd := exec.Command(
		"git",
		commit...,
	)
	cmd.Dir = args.PagesRepo.Checkout

	return runCommand(cmd)
}

func pushChanges(args *Args) error {
	cmd := repo.RemotePush(
		args.PagesRepo.Name,
		args.PagesRepo.Branch,
		args.PagesCommit.ForcePush,
		false,
	)
	cmd.Dir = args.PagesRepo.Checkout

	return runCommand(cmd)
}

func dirtyRepo(args *Args) bool {
	cmd := exec.Command(
		"git",
		"status",
		"--porcelain",
	)

	res := bytes.NewBufferString("")
	cmd.Dir = args.PagesRepo.Checkout
	cmd.Stdout = res
	cmd.Stderr = res

	err := runCommand(cmd)
	if err != nil {
		return false
	}

	if res.Len() > 0 {
		fmt.Fprintf(os.Stdout, "%s\n", res.String())

		return true
	}

	return false
}

func trace(cmd *exec.Cmd) {
	fmt.Fprintf(os.Stdout, "+ %s\n", strings.Join(cmd.Args, " "))
}

func runCommand(cmd *exec.Cmd) error {
	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
	}

	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}

	trace(cmd)

	return cmd.Run()
}

func pagesURL(args *Args) (*url.URL, error) {
	// See if a CNAME file is present
	if cname, err := os.ReadFile("CNAME"); err == nil {
		uri, errp := url.Parse(string(cname))
		if errp != nil {
			return nil, fmt.Errorf("could not parse link in cname file: %w", errp)
		}

		return uri, nil
	}

	// Determine url from repo information
	if args.Repo.Link == "" {
		return nil, fmt.Errorf("repo link not present: %w", errConfiguration)
	}

	uri, err := url.Parse(args.Repo.Link)
	if err != nil {
		return nil, fmt.Errorf("could not parse repo link: %w", err)
	}

	// Check for GitHub hosting
	if uri.Hostname() == "github.com" {
		pages, _ := url.Parse(fmt.Sprintf("https://%s.github.io", args.Repo.Namespace))

		// Check for organization page
		if pages.Hostname() == args.Repo.Name {
			return pages, nil
		}

		relPages, _ := url.Parse(fmt.Sprintf("./%s", args.Repo.Name))

		return pages.ResolveReference(relPages), nil
	}

	// Enterprise hosting
	uri.Path = ""
	relPages, _ := url.Parse(fmt.Sprintf("./pages/%s/%s", args.Repo.Namespace, args.Repo.Name))

	return uri.ResolveReference(relPages), nil
}
