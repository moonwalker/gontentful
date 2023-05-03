package github

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v48/github"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"

	"github.com/moonwalker/moonbase/internal/env"
	"github.com/moonwalker/moonbase/pkg/content"
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

	tree, resp, err := getCommitTree(ctx, githubClient, owner, repo, *reference.Object.SHA, []BlobEntry{
		{
			Path:    path,
			Content: content,
		}})

	resp, err = pushCommit(ctx, githubClient, reference, tree, owner, repo, commitMessage)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

type BlobEntry struct {
	Path    string
	Content *string
}

func getCommitTree(ctx context.Context, githubClient *github.Client, owner string, repo string, sha string, items []BlobEntry) (*github.Tree, *github.Response, error) {
	// Create a tree with what to commit.
	entries := make([]*github.TreeEntry, 0)
	for _, i := range items {
		entries = append(entries, &github.TreeEntry{
			Path:    github.String(i.Path),
			Type:    github.String("blob"),
			Mode:    github.String("100644"),
			Content: i.Content,
		})
	}

	return githubClient.Git.CreateTree(ctx, owner, repo, sha, entries)
}

func CommitBlobs(ctx context.Context, accessToken string, owner string, repo string, ref string, items []BlobEntry, commitMessage string) (*github.Response, error) {
	githubClient := ghClient(ctx, accessToken)

	reference, resp, err := githubClient.Git.GetRef(ctx, owner, repo, "refs/heads/"+ref)
	if err != nil {
		return resp, err
	}

	tree, resp, err := getCommitTree(ctx, githubClient, owner, repo, *reference.Object.SHA, items)
	if err != nil {
		return resp, err
	}

	resp, err = pushCommit(ctx, githubClient, reference, tree, owner, repo, commitMessage)
	if err != nil {
		return resp, err
	}

	return resp, nil
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

func DeleteFiles(ctx context.Context, accessToken string, owner string, repo string, ref string, path string, commitMessage string, fileNames []string) (*github.Response, error) {
	githubClient := ghClient(ctx, accessToken)

	_, rc, resp, err := githubClient.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{
		Ref: ref,
	})
	if err != nil {
		return resp, err
	}

	fn := make(map[string]bool)
	for _, f := range fileNames {
		fn[f] = true
	}

	items := make([]BlobEntry, 0)
	for _, c := range rc {
		if *c.Type == "file" && fn[*c.Name] {
			items = append(items, BlobEntry{
				Path:    *c.Path,
				Content: nil,
			})

		}
	}

	resp, err = CommitBlobs(ctx, accessToken, owner, repo, ref, items, commitMessage)
	return resp, err
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

func GetContentsRecursive_old(ctx context.Context, accessToken string, owner string, repo string, ref string, path string) ([]*github.RepositoryContent, *github.Response, error) {
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

func GetArchivedContents(ctx context.Context, accessToken string, owner string, repo string, ref string, path string) ([]*github.RepositoryContent, *github.Response, error) {
	githubClient := ghClient(ctx, accessToken)

	url, resp, err := githubClient.Repositories.GetArchiveLink(ctx, owner, repo, github.Tarball, nil, false)
	if err != nil {
		return nil, resp, err
	}

	res, err := http.Get(url.String())
	if err != nil {
		return nil, resp, err
	}
	defer res.Body.Close()

	dir, err := os.MkdirTemp("", "git-archive")
	if err != nil {
		return nil, resp, err
	}
	defer os.RemoveAll(dir)

	rcs := make([]*github.RepositoryContent, 0)

	uncompressedStream, err := gzip.NewReader(res.Body)
	if err != nil {
		return nil, resp, fmt.Errorf("gzip NewReader failed: %s", err.Error())
	}

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, resp, fmt.Errorf("tarReader Next() failed: %s", err.Error())
		}

		if header.Typeflag == tar.TypeReg {
			fi := strings.Index(header.Name, "/")
			li := strings.LastIndex(header.Name, "/")
			path := header.Name[fi:]
			name := header.Name[li+1:]
			bs, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return nil, resp, fmt.Errorf("tarReader ReadAll() failed: %s", err.Error())
			}
			content := string(bs)
			rcs = append(rcs, &github.RepositoryContent{
				Name:    &name,
				Path:    &path,
				Content: &content,
			})
		}
	}

	return rcs, resp, nil
}

func GetContentsRecursive(ctx context.Context, accessToken string, owner string, repo string, ref string, path string) ([]*github.RepositoryContent, *github.Response, error) {
	githubClient := ghClient(ctx, accessToken)

	_, rc, resp, err := githubClient.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{
		Ref: ref,
	})
	if err != nil {
		return nil, resp, err
	}

	rcs := make([]*github.RepositoryContent, 0)
	for _, c := range rc {
		if *c.Type == "file" {
			b, err := downloadFile(*c.DownloadURL)
			if err != nil {
				return nil, resp, err
			}
			content := string(b)
			if strings.Contains(content, "Not found") {
				log.Fatal("couldn't download file")
			}
			c.Content = &content
			rcs = append(rcs, c)
		}
		if *c.Type == "dir" {
			rcr, _, err := GetContentsRecursive(ctx, accessToken, owner, repo, ref, *c.Path)
			if err != nil {
				return nil, resp, err
			}
			rcs = append(rcs, rcr...)
		}
	}

	return rcs, resp, nil
}

func GetAllLocaleContents(ctx context.Context, accessToken string, owner string, repo string, ref string, path string, prefix string) ([]*github.RepositoryContent, *github.Response, error) {
	githubClient := ghClient(ctx, accessToken)

	_, rc, resp, err := githubClient.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{
		Ref: ref,
	})
	if err != nil {
		return nil, resp, err
	}

	rcs := make([]*github.RepositoryContent, 0)
	for _, c := range rc {
		if *c.Type == "file" && strings.HasPrefix(*c.Name, prefix) {
			b, err := downloadFile(*c.DownloadURL)
			if err != nil {
				return nil, resp, err
			}
			content := string(b)
			if strings.Contains(content, "Not found") {
				log.Fatal("couldn't download file")
			}
			c.Content = &content
			rcs = append(rcs, c)
		}
	}

	return rcs, resp, nil
}

func downloadFile(downloadURL string) ([]byte, error) {
	resp, err := http.Get(downloadURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func GetSchemasRecursive(ctx context.Context, accessToken string, owner string, repo string, ref string, path string) ([]*github.RepositoryContent, *github.Response, error) {
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
		if *te.Type == "blob" && strings.HasSuffix(*te.Path, content.JsonSchemaName) {
			rc, _, resp, err := githubClient.Repositories.GetContents(ctx, owner, repo, filepath.Join(path, *te.Path), &github.RepositoryContentGetOptions{})
			if err != nil {
				return nil, resp, err
			}
			rcs = append(rcs, rc)
		}
	}
	return rcs, resp, nil
}
