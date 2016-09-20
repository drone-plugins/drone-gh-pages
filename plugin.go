package main

import (
	"bytes"
	"fmt"

	"os"
	"os/exec"

	"path/filepath"

	"github.com/drone-plugins/drone-git-push/repo"
)

type (
	Repo struct {
		Clone string
	}

	Build struct {
		Path string
	}

	Author struct {
		Name  string
		Email string
	}

	Commit struct {
		Author Author
	}

	Netrc struct {
		Machine  string
		Login    string
		Password string
	}

	Config struct {
		Key            string
		UpstreamName   string
		TargetBranch   string
		TemporaryBase  string
		PagesDirectory string
	}

	Plugin struct {
		Repo   Repo
		Build  Build
		Commit Commit
		Netrc  Netrc
		Config Config
	}
)

func (p Plugin) Exec() error {
	cloneUrl := p.Repo.Clone

	upstreamName := p.Config.UpstreamName
	targetBranch := p.Config.TargetBranch
	temporaryBase := p.Config.TemporaryBase
	pagesDirectory := p.Config.PagesDirectory
	buildPath := p.Build.Path

	netrcMachine := p.Netrc.Machine
	netrcLogin := p.Netrc.Login
	netrcPassword := p.Netrc.Password
	sshKey := p.Config.Key

	userName := p.Commit.Author.Name
	userEmail := p.Commit.Author.Email

	temporaryBaseDirectory := ""

	if filepath.IsAbs(temporaryBase) {
		temporaryBaseDirectory = temporaryBase
	} else {
		temporaryBaseDirectory = filepath.Join(buildPath, temporaryBase)
	}

	temporaryPagesDirectory := filepath.Join(temporaryBaseDirectory, pagesDirectory)
	fullPagesDirectory := filepath.Join(buildPath, pagesDirectory)

	// generate the .netrc file
	if err := repo.WriteNetrc(netrcMachine, netrcLogin, netrcPassword); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}

	// write the rsa private key if provided
	if err := repo.WriteKey(sshKey); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}

	if err := repo.GlobalUser(userEmail).Run(); err != nil {
		return err
	}

	if err := repo.GlobalName(userName).Run(); err != nil {
		return err
	}

	if err := os.MkdirAll(temporaryPagesDirectory, 0777); err != nil {
		return err
	}

	defer os.RemoveAll(temporaryPagesDirectory)

	err := runPublishSteps(
		buildPath,
		temporaryBaseDirectory,
		temporaryPagesDirectory,
		fullPagesDirectory,
		targetBranch,
		upstreamName,
		cloneUrl,
	)

	if err != nil {
		return err
	}

	return nil
}

func runCommand(cmd *exec.Cmd, out *bytes.Buffer) error {
	if out == nil {
		if cmd.Stdout == nil {
			cmd.Stdout = os.Stdout
		}
	} else {
		cmd.Stdout = out
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error: %+v %s\n", cmd, err)
		return err
	}

	return nil
}

func runPublishSteps(
	workspacePath string,
	temporaryBaseDirectory string,
	temporaryPagesDirectory string,
	fullPagesDirectory string,
	vargsTargetBranch string,
	vargsUpstreamName string,
	cloneUrl string) error {

	// Get the commit message
	msgRaw, err := commitMessage(workspacePath).Output()

	if err != nil {
		return err
	}

	msg := fmt.Sprintf("%s", msgRaw)
	fmt.Printf("%s\n", msg)

	// Set up git config (rsa, etc)
	fmt.Printf("about to clone\n")
	err = runCommand(cloneTarget(workspacePath, vargsTargetBranch, cloneUrl, temporaryPagesDirectory), nil)
	if err != nil {
		return err
	}

	fmt.Printf("about to rsync\n")
	err = runCommand(rsyncPages(workspacePath, fullPagesDirectory, temporaryBaseDirectory), nil)
	if err != nil {
		return err
	}

	fmt.Printf("about to add to clone\n")
	var addResult bytes.Buffer
	err = runCommand(addTemporaryFilesToClone(temporaryPagesDirectory), &addResult)
	if err != nil {
		fmt.Printf("Error on add: %s", addResult.String())
		return err
	}

	// For now, if the add resulted in a success, with output, we are assuming that there are
	// changes to commit and push
	if addResult.Len() > 0 {
		fmt.Printf("about to commit\n")
		err = runCommand(commitTemporaryFilesToClone(temporaryPagesDirectory, msg), nil)
		if err != nil {
			return err
		}

		fmt.Printf("about to push\n")
		err = runCommand(pushTemporaryClone(temporaryPagesDirectory, vargsUpstreamName, vargsTargetBranch), nil)
		if err != nil {
			return err
		}
	}

	return nil
}

// Returns command to get the commit message for the gh-pages
// commit based on the last commit in the build
func commitMessage(workspacePath string) *exec.Cmd {
	cmd := exec.Command(
		"git",
		"show",
		"-q",
	)
	cmd.Dir = workspacePath
	return cmd
}

// Returns command to clone gh-pages into our temporary location.
// git clone -b gh-pages --single-branch repo .tmp/[pages directory]
func cloneTarget(workspacePath string, targetBranch string, repo string, temporaryPagesDirectory string) *exec.Cmd {
	cmd := exec.Command(
		"git",
		"clone",
		"-b",
		targetBranch,
		"--single-branch",
		repo,
		temporaryPagesDirectory,
	)
	cmd.Dir = workspacePath
	return cmd
}

// Copy the pages content to the temporary location
// rsync --delete --exclude .git -r docs .tmp
func rsyncPages(workspacePath string, pagesDirectory string, temporaryBaseDirectory string) *exec.Cmd {
	cmd := exec.Command(
		"rsync",
		"--delete",
		"--exclude",
		".git",
		"-r",
		pagesDirectory,
		temporaryBaseDirectory,
	)
	cmd.Dir = workspacePath
	return cmd
}

// Add the files in the temporary directory to the commit
// we want to make in the clone.
func addTemporaryFilesToClone(temporaryPagesDirectory string) *exec.Cmd {
	cmd := exec.Command(
		"git",
		"add",
		"--verbose",
		".",
	)
	cmd.Dir = temporaryPagesDirectory
	return cmd
}

// Commit the working version of the pages content to
// our clone
func commitTemporaryFilesToClone(temporaryPagesDirectory string, message string) *exec.Cmd {
	cmd := exec.Command(
		"git",
		"commit",
		"-m",
		message,
	)
	cmd.Dir = temporaryPagesDirectory
	return cmd
}

// Push our clone to the upstream
func pushTemporaryClone(temporaryPagesDirectory string, upstreamName string, targetBranch string) *exec.Cmd {
	cmd := repo.RemotePush(upstreamName, targetBranch, false)
	cmd.Dir = temporaryPagesDirectory
	return cmd
}
