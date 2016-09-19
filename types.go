package main

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
)
