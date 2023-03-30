package github

type Owner struct {
	Name string `json:"name"`
}

type Repository struct {
	Name         string `json:"name"`
	FullName     string `json:"full_name"`
	URL          string `json:"url"`
	Owner        Owner  `json:"owner"`
	MasterBranch string `json:"master_branch"`
}

type Commit struct {
	Added    []string `json:"added"`
	Removed  []string `json:"removed"`
	Modified []string `json:"modified"`
}

type PushHookPayload struct {
	Ref        string     `json:"ref"`
	Repository Repository `json:"repository"`
	Commit     Commit     `json:"head_commit"`
}
