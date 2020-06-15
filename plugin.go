package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/appleboy/drone-git-push/repo"
	"github.com/pkg/errors"
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
		WorkDirectory  string
		ExcludeCname   bool
		Delete         bool
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
	if err := p.prepare(); err != nil {
		return err
	}

	if err := p.process(); err != nil {
		return err
	}

	return nil
}

func (p Plugin) prepare() error {
	if p.Netrc.Login != "" {
		if err := repo.WriteNetrc(p.Netrc.Machine, p.Netrc.Login, p.Netrc.Password); err != nil {
			return errors.Wrap(err, "failed to write netrc")
		}
	}

	if p.Config.Key != "" {
		if err := repo.WriteKey(p.Config.Key); err != nil {
			return errors.Wrap(err, "failed to write sshkey")
		}
	}

	if err := repo.GlobalUser(p.Commit.Author.Email).Run(); err != nil {
		return errors.Wrap(err, "failed to set email")
	}

	if err := repo.GlobalName(p.Commit.Author.Name).Run(); err != nil {
		return errors.Wrap(err, "failed to set author")
	}

	return nil
}

func (p Plugin) process() error {
	defer os.RemoveAll(p.Config.TemporaryBase)

	if err := p.cloneTarget(); err != nil {
		return errors.Wrap(err, "failed to clone target")
	}

	if err := p.rsyncPages(); err != nil {
		return errors.Wrap(err, "failed to sync pages")
	}

	if p.dirtyRepo() {
		if err := p.stageChanges(); err != nil {
			return errors.Wrap(err, "failed to stage changes")
		}

		if err := p.commitChanges(); err != nil {
			return errors.Wrap(err, "failed to commit changes")
		}

		if err := p.pushChanges(); err != nil {
			return errors.Wrap(err, "failed to push changes")
		}
	}

	return nil
}

func (p Plugin) cloneTarget() error {
	cmd := exec.Command(
		"git",
		"clone",
		"-b",
		p.Config.TargetBranch,
		"--single-branch",
		p.Repo.Clone,
		p.Config.WorkDirectory,
	)

	cmd.Dir = p.Build.Path
	return runCommand(cmd)
}

func (p Plugin) rsyncPages() error {
	args := []string{
		"-r",
		"--exclude",
		".git",
	}

	if p.Config.ExcludeCname {
		args = append(
			args,
			"--exclude",
			"CNAME",
		)
	}

	if p.Config.Delete {
		args = append(
			args,
			"--delete",
		)
	}

	args = append(
		args,
		p.Config.PagesDirectory,
		p.Config.TemporaryBase,
	)

	cmd := exec.Command(
		"rsync",
		args...,
	)

	cmd.Dir = p.Build.Path
	return runCommand(cmd)
}

func (p Plugin) stageChanges() error {
	cmd := exec.Command(
		"git",
		"add",
		".",
	)

	cmd.Dir = p.Config.WorkDirectory
	return runCommand(cmd)
}

func (p Plugin) commitChanges() error {
	message, err := p.commitMessage()

	if err != nil {
		return err
	}

	cmd := exec.Command(
		"git",
		"commit",
		"-m",
		string(message),
	)

	cmd.Dir = p.Config.WorkDirectory
	return runCommand(cmd)
}

func (p Plugin) pushChanges() error {
	cmd := repo.RemotePush(
		p.Config.UpstreamName,
		p.Config.TargetBranch,
		false,
		false,
	)

	cmd.Dir = p.Config.WorkDirectory
	return runCommand(cmd)
}

func (p Plugin) dirtyRepo() bool {
	cmd := exec.Command(
		"git",
		"status",
		"--porcelain",
	)

	res := bytes.NewBufferString("")
	cmd.Dir = p.Config.WorkDirectory
	cmd.Stdout = res
	cmd.Stderr = res

	err := runCommand(cmd)

	if err != nil {
		return false
	}

	if res.Len() > 0 {
		fmt.Print(res.String())
		return true
	}

	return false
}

func (p Plugin) commitMessage() ([]byte, error) {
	cmd := exec.Command(
		"git",
		"show",
		"-q",
	)

	cmd.Dir = p.Build.Path
	return cmd.Output()
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
