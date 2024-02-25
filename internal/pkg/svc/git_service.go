package svc

import (
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/hhatto/gocloc"
)

type Repo struct {
	gitRemote      string
	localDirectory string
	repositoryName string
	githubUser     string
	githubToken    string
}

func NewRepo(url, dir, name, ghUser, ghToken string) *Repo {
	return &Repo{
		gitRemote:      url,
		localDirectory: dir,
		repositoryName: name,
		githubUser:     ghUser,
		githubToken:    ghToken,
	}
}

// Clone godoc
func (r *Repo) Clone() error {
	auth := &http.BasicAuth{
		Username: r.githubUser,
		Password: r.githubToken,
	}
	_, err := git.PlainClone(r.localDirectory, false, &git.CloneOptions{
		URL:      r.gitRemote,
		Progress: os.Stdout,
		Auth:     auth,
	})
	if err != nil {
		return err
	}

	return nil
}

// Delete godoc
func (r *Repo) Delete() error {
	err := os.RemoveAll(r.localDirectory)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repo) SelfWrittenLOC() (int, error) {
	languages := gocloc.NewDefinedLanguages()
	options := gocloc.NewClocOptions()

	paths := []string{
		r.localDirectory,
	}

	processor := gocloc.NewProcessor(languages, options)

	result, err := processor.Analyze(paths)
	if err != nil {
		return -1, nil
	}

	return int(result.Total.Code), nil
}

func (r *Repo) LibraryLOC() (int, error) {
	return -1, nil
}
