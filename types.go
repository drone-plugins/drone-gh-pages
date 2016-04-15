package main

type Params struct {
	UpstreamName   string `json:"remote"`
	PagesDirectory string `json:"source"`
	TemporaryBase  string `json:"temp"`
	TargetBranch   string `json:"branch"`
}
