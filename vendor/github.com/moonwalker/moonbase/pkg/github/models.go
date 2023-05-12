package github

import "time"

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
	ID        string    `json:"id"`
	Added     []string  `json:"added"`
	Removed   []string  `json:"removed"`
	Modified  []string  `json:"modified"`
	Timestamp time.Time `json:"timestamp"`
}

type PushHookPayload struct {
	Ref        string     `json:"ref"`
	Repository Repository `json:"repository"`
	Commit     Commit     `json:"head_commit"`
}
