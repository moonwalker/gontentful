package github

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
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
	t, err := ghConfig().Exchange(context.Background(), code)
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

func CreateBlob(ctx context.Context, accessToken string, owner string, repo string, ref string, content *string, encoding *string) (*github.Blob, *github.Response, error) {
	githubClient := ghClient(ctx, accessToken)

	blob, resp, err := githubClient.Git.CreateBlob(ctx, owner, repo, &github.Blob{
		Content:  content,
		Encoding: encoding,
	})

	return blob, resp, err
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
	if err != nil {
		return resp, err
	}

	resp, err = pushCommit(ctx, githubClient, reference, tree, owner, repo, commitMessage)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

type BlobEntry struct {
	Path    string
	Content *string
	SHA     *string
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
			SHA:     i.SHA,
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

	resp, items, err := getFolderContentRecursive(ctx, githubClient, owner, repo, ref, path)
	if err != nil {
		return resp, err
	}

	if len(items) > 0 {
		resp, err = CommitBlobs(ctx, accessToken, owner, repo, ref, items, commitMessage)
	}

	return resp, nil
}

func getFolderContentRecursive(ctx context.Context, githubClient *github.Client, owner string, repo string, ref string, path string) (*github.Response, []BlobEntry, error) {
	_, rc, resp, err := githubClient.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{
		Ref: ref,
	})
	if err != nil {
		return resp, nil, err
	}

	items := make([]BlobEntry, 0)
	for _, c := range rc {
		if *c.Type == "dir" {
			var folderItems []BlobEntry
			resp, folderItems, err = getFolderContentRecursive(ctx, githubClient, owner, repo, ref, *c.Path)
			items = append(items, folderItems...)
		} else {
			items = append(items, BlobEntry{
				Path:    *c.Path,
				Content: nil,
			})
		}
		if err != nil {
			return resp, nil, err
		}
	}

	return resp, items, nil
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

	if len(items) > 0 {
		resp, err = CommitBlobs(ctx, accessToken, owner, repo, ref, items, commitMessage)
	}
	return resp, err
}

func GetDeleteFileEntries(ctx context.Context, accessToken string, owner string, repo string, ref string, path string, commitMessage string, fileNames []string) ([]BlobEntry, error) {
	githubClient := ghClient(ctx, accessToken)

	_, rc, _, err := githubClient.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{
		Ref: ref,
	})
	if err != nil {
		return nil, err
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

	return items, nil
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

	var opt *github.RepositoryContentGetOptions
	if len(path) > 0 {
		dirSha, resp, err := getDirectorySha(ctx, githubClient, owner, repo, ref, path)
		if err != nil {
			return nil, resp, err
		}
		opt = &github.RepositoryContentGetOptions{
			Ref: dirSha,
		}
	}
	url, resp, err := githubClient.Repositories.GetArchiveLink(ctx, owner, repo, github.Tarball, opt, false)
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
			bs, err := io.ReadAll(tarReader)
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
		// not needed with the new content structure
		/*if *c.Type == "dir" {
			rcr, _, err := GetContentsRecursive(ctx, accessToken, owner, repo, ref, *c.Path)
			if err != nil {
				return nil, resp, err
			}
			rcs = append(rcs, rcr...)
		}*/
	}

	return rcs, resp, nil
}

func GetAllLocaleContents(ctx context.Context, accessToken string, owner string, repo string, ref string, path string) ([]*github.RepositoryContent, *github.Response, error) {
	githubClient := ghClient(ctx, accessToken)

	_, rc, resp, err := githubClient.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{
		Ref: ref,
	})
	if err != nil {
		return nil, resp, err
	}

	rcs := make([]*github.RepositoryContent, 0)
	for _, c := range rc {
		if *c.Type == "file" && (*c.Name == content.JsonSchemaName || filepath.Ext(*c.Name) == ".json") {
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

func GetAllLocaleContentsWithTree(ctx context.Context, accessToken string, owner string, repo string, ref string, path string, prefix string) ([]*github.RepositoryContent, *github.Response, error) {
	githubClient := ghClient(ctx, accessToken)
	sha, resp, err := getDirectorySha(ctx, githubClient, owner, repo, ref, path)
	if err != nil {
		return nil, resp, err
	}
	if sha == "" {
		sha = "main"
	}
	tree, resp, err := githubClient.Git.GetTree(ctx, owner, repo, sha, false)
	if err != nil {
		return nil, resp, err
	}

	rcs := make([]*github.RepositoryContent, 0)
	for _, te := range tree.Entries {
		if *te.Type == "blob" && (strings.HasPrefix(*te.Path, prefix) || strings.HasSuffix(*te.Path, content.JsonSchemaName)) {
			rc, _, resp, err := githubClient.Repositories.GetContents(ctx, owner, repo, filepath.Join(path, *te.Path), &github.RepositoryContentGetOptions{})
			if err != nil {
				return nil, resp, err
			}
			c, err := rc.GetContent()
			if err != nil {
				return nil, resp, err
			}
			rc.Content = &c
			rcs = append(rcs, rc)
		}
	}

	return rcs, resp, nil
}

func GetFilesContent(ctx context.Context, accessToken string, owner string, repo string, ref string, path string, files []string) ([]*github.RepositoryContent, *github.Response, error) {
	githubClient := ghClient(ctx, accessToken)
	var resp *github.Response

	rcs := make([]*github.RepositoryContent, 0)
	for _, fn := range files {
		fnWithPath := filepath.Join(path, fn)
		rc, _, resp, err := githubClient.Repositories.GetContents(ctx, owner, repo, fnWithPath, &github.RepositoryContentGetOptions{})
		if err != nil {
			return nil, resp, err
		}
		c, err := rc.GetContent()
		if err != nil {
			return nil, resp, err
		}
		rc.Content = &c
		rcs = append(rcs, rc)
	}

	return rcs, resp, nil
}

func SearchContentsByID(ctx context.Context, accessToken string, owner string, repo string, ref string, path string, id string) ([]*github.RepositoryContent, *github.Response, error) {
	githubClient := ghClient(ctx, accessToken)

	_, rc, resp, err := githubClient.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{
		Ref: ref,
	})
	if err != nil {
		return nil, resp, err
	}

	rcs := make([]*github.RepositoryContent, 0)
	for _, c := range rc {
		if *c.Type == "file" && *c.Name != content.JsonSchemaName {
			b, err := downloadFile(*c.DownloadURL)
			if err != nil {
				return nil, resp, err
			}
			content := string(b)
			if strings.Contains(content, "Not found") {
				log.Fatal("couldn't download file")
			}
			if strings.HasPrefix(content, fmt.Sprintf(`{"id":"%s",`, id)) {
				c.Content = &content
				rcs = append(rcs, c)
			}
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

	tree, resp, err := githubClient.Git.GetTree(ctx, owner, repo, sha, false)
	if err != nil {
		return nil, resp, err
	}

	rcs := make([]*github.RepositoryContent, 0)
	for _, te := range tree.Entries {
		if *te.Type == "tree" {
			subTree, resp, err := githubClient.Git.GetTree(ctx, owner, repo, *te.SHA, false)
			if err != nil {
				return nil, resp, err
			}
			tree.Entries = append(tree.Entries, subTree.Entries...)
		} else if *te.Type == "blob" && strings.HasSuffix(*te.Path, content.JsonSchemaName) {
			rc, _, resp, err := githubClient.Repositories.GetContents(ctx, owner, repo, filepath.Join(path, *te.Path), &github.RepositoryContentGetOptions{})
			if err != nil {
				return nil, resp, err
			}
			rcs = append(rcs, rc)
		}
	}
	return rcs, resp, nil
}
