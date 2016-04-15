package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"github.com/drone-plugins/drone-git-push/repo"
	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-plugin-go/plugin"
)

var (
	buildCommit string
)

var netrcFile = `
machine %s
login %s
password %s
`

func main() {
	fmt.Printf("Drone gh-pages deployment plugin built from %s\n", buildCommit)

	v := new(Params)
	r := new(drone.Repo)
	b := new(drone.Build)
	w := new(drone.Workspace)
	plugin.Param("repo", r)
	plugin.Param("build", b)
	plugin.Param("workspace", w)
	plugin.Param("vargs", &v)
	plugin.MustParse()
	err := publishDocs(r, b, w, v)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
}

func publishDocs(r *drone.Repo, b *drone.Build, w *drone.Workspace, v *Params) error {
	if v.UpstreamName == "" {
		v.UpstreamName = "origin"
	}
	if v.TargetBranch == "" {
		v.TargetBranch = "gh-pages"
	}
	if v.TemporaryBase == "" {
		v.TemporaryBase = ".tmp"
	}
	if v.PagesDirectory == "" {
		v.PagesDirectory = "docs"
	}

	temporaryBaseDirectory := ""
	if filepath.IsAbs(v.TemporaryBase) {
		temporaryBaseDirectory = v.TemporaryBase
	} else {
		temporaryBaseDirectory = filepath.Join(w.Path, v.TemporaryBase)
	}
	temporaryPagesDirectory := filepath.Join(temporaryBaseDirectory, v.PagesDirectory)

	fullPagesDirectory := filepath.Join(w.Path, v.PagesDirectory)

	// generate the .netrc file
	if err := writeNetrc(w); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}

	// write the rsa private key if provided
	if err := repo.WriteKey(w); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}

	err := repo.GlobalUser(b).Run()
	if err != nil {
		return err
	}

	err = repo.GlobalName(b).Run()
	if err != nil {
		return err
	}

	err = os.MkdirAll(temporaryPagesDirectory, 0777)
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporaryPagesDirectory)

	err = runPublishSteps(w.Path, temporaryBaseDirectory, temporaryPagesDirectory, fullPagesDirectory, v.TargetBranch, v.UpstreamName, r.Clone)
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

	err := cmd.Run()
	if err != nil {
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

// Writes the netrc file.
func writeNetrc(in *drone.Workspace) error {
	if in.Netrc == nil || len(in.Netrc.Machine) == 0 {
		return nil
	}
	out := fmt.Sprintf(
		netrcFile,
		in.Netrc.Machine,
		in.Netrc.Login,
		in.Netrc.Password,
	)
	home := "/root"
	u, err := user.Current()
	if err == nil {
		home = u.HomeDir
	}
	path := filepath.Join(home, ".netrc")
	return ioutil.WriteFile(path, []byte(out), 0600)
}
