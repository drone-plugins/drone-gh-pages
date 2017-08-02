package main

import (
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/joho/godotenv"
	"github.com/urfave/cli"
)

var build = "0"

func main() {
	app := cli.NewApp()
	app.Name = "gh-pages plugin"
	app.Usage = "gh-pages plugin"
	app.Action = run
	app.Version = fmt.Sprintf("1.0.%s", build)
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "commit.author.name",
			Usage:  "git author name",
			EnvVar: "PLUGIN_USER_NAME,DRONE_COMMIT_AUTHOR",
		},
		cli.StringFlag{
			Name:   "commit.author.email",
			Usage:  "git author email",
			EnvVar: "PLUGIN_USER_EMAIL,DRONE_COMMIT_AUTHOR_EMAIL",
		},
		cli.StringFlag{
			Name:   "remote",
			Usage:  "git remote url",
			EnvVar: "DRONE_REMOTE_URL",
		},
		cli.StringFlag{
			Name:   "path",
			Usage:  "git clone path",
			EnvVar: "DRONE_WORKSPACE",
		},
		cli.StringFlag{
			Name:   "netrc.machine",
			Usage:  "netrc machine",
			EnvVar: "DRONE_NETRC_MACHINE",
		},
		cli.StringFlag{
			Name:   "netrc.username",
			Usage:  "netrc username",
			EnvVar: "DRONE_NETRC_USERNAME",
		},
		cli.StringFlag{
			Name:   "netrc.password",
			Usage:  "netrc password",
			EnvVar: "DRONE_NETRC_PASSWORD",
		},
		cli.StringFlag{
			Name:   "upstream-name",
			Usage:  "git upstream to target",
			EnvVar: "PLUGIN_UPSTREAM_NAME",
			Value:  "origin",
		},
		cli.StringFlag{
			Name:   "target-branch",
			Usage:  "git branch to target",
			EnvVar: "PLUGIN_TARGET_BRANCH",
			Value:  "gh-pages",
		},
		cli.StringFlag{
			Name:   "temporary-base",
			Usage:  "temporary directory for pages pull",
			EnvVar: "PLUGIN_TEMPORARY_BASE",
			Value:  ".tmp",
		},
		cli.StringFlag{
			Name:   "pages-directory",
			Usage:  "directory of content to publish",
			EnvVar: "PLUGIN_PAGES_DIRECTORY",
			Value:  "docs",
		},
		cli.StringFlag{
			Name:   "ssh-key",
			Usage:  "private ssh key",
			EnvVar: "PLUGIN_SSH_KEY,GIT_PUSH_SSH_KEY,SSH_KEY",
		},
		cli.StringFlag{
			Name:  "env-file",
			Usage: "source env file",
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}

}

func run(c *cli.Context) error {
	if c.String("env-file") != "" {
		_ = godotenv.Load(c.String("env-file"))
	}

	plugin := Plugin{
		Repo: Repo{
			Clone: c.String("remote"),
		},

		Build: Build{
			Path: c.String("path"),
		},

		Commit: Commit{
			Author: Author{
				Name:  c.String("commit.author.name"),
				Email: c.String("commit.author.email"),
			},
		},

		Netrc: Netrc{
			Login:    c.String("netrc.username"),
			Machine:  c.String("netrc.machine"),
			Password: c.String("netrc.password"),
		},
		Config: Config{
			Key:            c.String("ssh-key"),
			UpstreamName:   c.String("upstream-name"),
			TargetBranch:   c.String("target-branch"),
			TemporaryBase:  c.String("temporary-base"),
			PagesDirectory: c.String("pages-directory"),
		},
	}

	return plugin.Exec()
}
