package svc

import (
	"github.com/go-git/go-git/v5"
	"os"
)

type Repo struct {
	url  string
	dir  string
	name string
}

func NewRepo(url, dir, name string) *Repo {
	return &Repo{
		url:  url,
		dir:  dir,
		name: name,
	}
}

// Clone godoc
func (r *Repo) Clone() error {
	_, err := git.PlainClone(r.dir, false, &git.CloneOptions{
		URL:      r.url,
		Progress: os.Stdout,
	})
	if err != nil {
		return err
	}

	return nil
}

// Delete godoc
func (r *Repo) Delete() error {
	err := os.RemoveAll(r.dir)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repo) SelfWrittenLOC() (int, error) {
	return -1, nil
}

func (r *Repo) LibraryLOC() (int, error) {
	return -1, nil
}
