package main

import "testing"
import "bytes"

func TestCommitAppendToMessage(t *testing.T) {
	expectedSuffix := "[omg]"

	p := &Plugin{
		Repo: Repo{
			Clone: "remote",
		},

		Build: Build{
			Path: ".",
		},

		Commit: Commit{
			Author: Author{
				Name:  "commit.author.name",
				Email: "commit.author.email",
			},
			AppendToMessage: expectedSuffix,
		},

		Netrc: Netrc{
			Login:    "netrc.username",
			Machine:  "netrc.machine",
			Password: "netrc.password",
		},
		Config: Config{
			Key:            "ssh-key",
			UpstreamName:   "upstream-name",
			TargetBranch:   "target-branch",
			TemporaryBase:  "temporary-base",
			PagesDirectory: "pages-directory",
			ExcludeCname:   false,
			Delete:         false,
			ForcePush:      false,
		},
	}

	actual, err := p.commitMessage()
	if err != nil {
		t.Errorf("Exec should not error")
	}

	if !bytes.HasSuffix(actual, []byte(expectedSuffix)) {
		t.Errorf("Commit message did not end with expected suffix [%s], got: [%s]", expectedSuffix, string(actual))
	}
}
