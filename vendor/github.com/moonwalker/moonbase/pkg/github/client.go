package github

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/google/go-github/v48/github"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"

	"github.com/moonwalker/moonbase/internal/env"
)

var (
	ghScopes = []string{"user:email", "read:org", "repo"}
)

func ghConfig() *oauth2.Config {
	return &oauth2.Config{
		Scopes:       ghScopes,
		Endpoint:     githuboauth.Endpoint,
		ClientID:     env.GithubClientID,
		ClientSecret: env.GithubClientSecret,
	}
}

func ghClient(ctx context.Context, accessToken string) *github.Client {
	oauthClient := ghConfig().Client(ctx, &oauth2.Token{AccessToken: accessToken})
	return github.NewClient(oauthClient)
}

func AuthCodeURL(state string) string {
	return ghConfig().AuthCodeURL(state, oauth2.AccessTypeOnline)
}

func Exchange(code string) (string, error) {
	t, err := ghConfig().Exchange(oauth2.NoContext, code)
	if err != nil {
		return "", err
	}
	return t.AccessToken, nil
}

func GetUser(ctx context.Context, accessToken string) (*github.User, error) {
	user, _, err := ghClient(ctx, accessToken).Users.Get(ctx, "")
	return user, err
}

func ListRepositories(ctx context.Context, accessToken string, page, perPage int, sort, direction string) ([]*github.Repository, *github.Response, error) {
	repos, resp, err := ghClient(ctx, accessToken).Repositories.List(ctx, "", &github.RepositoryListOptions{
		Sort:      sort,
		Direction: direction,
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	})
	return repos, resp, err
}

func ListBranches(ctx context.Context, accessToken string, owner, repo string) ([]*github.Branch, *github.Response, error) {
	branches, resp, err := ghClient(ctx, accessToken).Repositories.ListBranches(ctx, owner, repo, &github.BranchListOptions{})
	return branches, resp, err
}

func GetTree(ctx context.Context, accessToken string, owner string, repo string, branch string, path string) ([]*github.RepositoryContent, *github.Response, error) {
	_, rc, resp, err := ghClient(ctx, accessToken).Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{
		Ref: branch,
	})
	if err != nil {
		return nil, resp, err
	}
	return rc, resp, nil
}

func GetFileContent(ctx context.Context, accessToken string, owner string, repo string, ref, path string) (*github.RepositoryContent, *github.Response, error) {
	if len(path) == 0 {
		return nil, nil, errors.New("path not provided")
	}

	fc, _, resp, err := ghClient(ctx, accessToken).Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{
		Ref: ref,
	})
	if err != nil {
		return nil, resp, err
	}

	return fc, resp, nil
}

func GetBlob(ctx context.Context, accessToken string, owner string, repo string, ref, path string) ([]byte, *github.Response, error) {
	fc, resp, err := GetFileContent(ctx, accessToken, owner, repo, ref, path)
	if err != nil {
		return nil, resp, err
	}

	decodedBlob, err := fc.GetContent()
	if err != nil {
		return nil, resp, err
	}

	return []byte(decodedBlob), resp, nil
}

func CommitBlob(ctx context.Context, accessToken string, owner string, repo string, ref string, path string, content *string, commitMessage string) (*github.Response, error) {
	githubClient := ghClient(ctx, accessToken)

	reference, resp, err := githubClient.Git.GetRef(ctx, owner, repo, "refs/heads/"+ref)
	if err != nil {
		return resp, err
	}

	tree, resp, err := getCommitTree(ctx, githubClient, owner, repo, *reference.Object.SHA, path, content)
	if err != nil {
		return resp, err
	}

	resp, err = pushCommit(ctx, githubClient, reference, tree, owner, repo, commitMessage)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// getCommitTree generates the tree to commit based on the given files and the commit of the ref
func getCommitTree(ctx context.Context, githubClient *github.Client, owner string, repo string, sha string, path string, content *string) (*github.Tree, *github.Response, error) {
	// Create a tree with what to commit.
	entries := []*github.TreeEntry{
		{
			Path:    github.String(path),
			Type:    github.String("blob"),
			Mode:    github.String("100644"),
			Content: content,
		},
	}

	return githubClient.Git.CreateTree(ctx, owner, repo, sha, entries)
}

// pushCommit creates the commit in the given reference using the given tree
func pushCommit(ctx context.Context, githubClient *github.Client, ref *github.Reference, tree *github.Tree, owner string, repo string, commitMessage string) (*github.Response, error) {
	// Get the parent commit
	parent, resp, err := githubClient.Repositories.GetCommit(ctx, owner, repo, *ref.Object.SHA, nil)
	if err != nil {
		return resp, err
	}
	parent.Commit.SHA = parent.SHA

	commit := &github.Commit{Message: &commitMessage, Tree: tree, Parents: []*github.Commit{parent.Commit}}
	newCommit, resp, err := githubClient.Git.CreateCommit(ctx, owner, repo, commit)
	if err != nil {
		return resp, err
	}

	// Attach the commit to the branch
	ref.Object.SHA = newCommit.SHA
	_, resp, err = githubClient.Git.UpdateRef(ctx, owner, repo, ref, false)
	if err != nil {
		return resp, err
	}

	// Crreate pull request if needed
	/*rep, resp, err := githubClient.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return resp, err
	}

	branch := (*ref.Ref)[strings.LastIndex(*ref.Ref, "/")+1:]
	if branch != *rep.DefaultBranch {
		resp, err = createPR(ctx, githubClient, owner, repo, *ref.Ref, *rep.DefaultBranch)
		if err != nil {
			return resp, err
		}
	}*/

	return resp, err
}

// createPR creates a pull request
func createPR(ctx context.Context, githubClient *github.Client, owner string, repo string, branch string, baseBranch string) (*github.Response, error) {
	subject := "pr"
	newPR := &github.NewPullRequest{
		Title:               &subject,
		Head:                &branch,
		Base:                &baseBranch,
		MaintainerCanModify: github.Bool(true),
	}

	_, resp, err := githubClient.PullRequests.Create(ctx, owner, repo, newPR)
	if err != nil {
		return nil, err
	}

	return resp, err
}

func DeleteFolder(ctx context.Context, accessToken string, owner string, repo string, ref string, path string, commitMessage string) (*github.Response, error) {
	githubClient := ghClient(ctx, accessToken)

	_, rc, resp, err := githubClient.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{
		Ref: ref,
	})
	if err != nil {
		return resp, err
	}

	for _, c := range rc {
		if *c.Type == "dir" {
			resp, err = DeleteFolder(ctx, accessToken, owner, repo, ref, *c.Path, commitMessage)
		} else {
			_, _, err = githubClient.Repositories.DeleteFile(ctx, owner, repo, *c.Path, &github.RepositoryContentFileOptions{
				Message: &commitMessage,
				SHA:     c.SHA,
				Branch:  &ref,
			})

		}
		if err != nil {
			return resp, err
		}
	}

	return resp, nil
}

func GetCommits(ctx context.Context, accessToken string, owner string, repo string, ref string) ([]*github.RepositoryCommit, *github.Response, error) {
	rc, resp, err := ghClient(ctx, accessToken).Repositories.ListCommits(ctx, owner, repo, &github.CommitsListOptions{
		SHA: ref,
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: 10,
		},
	})
	if err != nil {
		return nil, resp, err
	}

	return rc, resp, nil
}

func GetDirectorySha(ctx context.Context, accessToken string, owner string, repo string, ref string, path string) string {
	githubClient := ghClient(ctx, accessToken)
	sha, _, _ := getDirectorySha(ctx, githubClient, owner, repo, ref, path)
	return sha
}

func getDirectorySha(ctx context.Context, githubClient *github.Client, owner string, repo string, ref string, path string) (string, *github.Response, error) {
	if len(path) > 0 {
		parentPath := filepath.Dir(path)
		_, dc, resp, err := githubClient.Repositories.GetContents(ctx, owner, repo, parentPath, &github.RepositoryContentGetOptions{
			Ref: ref,
		})
		if err != nil {
			return "", resp, err
		}

		for _, te := range dc {
			if *te.Type == "dir" && *te.Path == path {
				return *te.SHA, resp, err
			}
		}
		return "", resp, err
	}

	br, resp, err := githubClient.Repositories.GetBranch(ctx, owner, repo, ref, false)
	if err != nil {
		return "", resp, err
	}

	if br.Commit != nil {
		return *br.Commit.SHA, resp, err
	}
	return "", resp, err
}

func GetContentsRecursive(ctx context.Context, accessToken string, owner string, repo string, ref string, path string) ([]*github.RepositoryContent, *github.Response, error) {
	githubClient := ghClient(ctx, accessToken)

	sha, resp, err := getDirectorySha(ctx, githubClient, owner, repo, ref, path)
	if err != nil {
		return nil, resp, err
	}

	if sha == "" {
		sha = "main"
	}
	tree, resp, err := githubClient.Git.GetTree(ctx, owner, repo, sha, true)
	if err != nil {
		return nil, resp, err
	}

	rcs := make([]*github.RepositoryContent, 0)
	for _, te := range tree.Entries {
		if *te.Type == "blob" {
			rc, _, resp, err := githubClient.Repositories.GetContents(ctx, owner, repo, filepath.Join(path, *te.Path), &github.RepositoryContentGetOptions{})
			if err != nil {
				return nil, resp, err
			}
			rcs = append(rcs, rc)
		}
	}

	return rcs, resp, nil
}
