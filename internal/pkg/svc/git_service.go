// Package svc provides services required from the application.
package svc

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/haapjari/glass-api/internal/pkg/logger"
	"github.com/haapjari/glass-api/internal/pkg/utils"
	"github.com/hhatto/gocloc"
)

// Repo represents a structure of a GitHub Repository, and offers metadata and
// functionality.
type Repo struct {
	remote         string
	dir            string
	repositoryName string
	user           string
	token          string
	tasks          chan string

	log logger.Logger
}

// NewRepo is a constructor.
func NewRepo(url, dir, name, user, token string, logger logger.Logger) *Repo {
	r := &Repo{
		remote:         url,
		dir:            dir,
		repositoryName: name,
		user:           user,
		token:          token,
		tasks:          make(chan string),
		log:            logger,
	}

	return r
}

// Clone is a wrapper for "git clone" command.
func (r *Repo) Clone() error {
	auth := &http.BasicAuth{
		Username: r.user,
		Password: r.token,
	}

	_, err := git.PlainClone(r.dir, false, &git.CloneOptions{
		URL:      r.remote,
		Progress: os.Stdout,
		Auth:     auth,
	})
	if err != nil {
		return err
	}

	return nil
}

// Delete is a wrapper for unix command "rm".
func (r *Repo) Delete() error {
	if err := os.RemoveAll(r.dir); err != nil {
		return err
	}

	return nil
}

// SelfWrittenLOC godoc.
func (r *Repo) SelfWrittenLOC() (int, error) {
	return r.calcLOC(r.dir)
}

// CalculateLOC is a function that calculates LOC of "Go" code within a directory.
func (r *Repo) calcLOC(dir string) (int, error) {
	languages := gocloc.NewDefinedLanguages()
	options := gocloc.NewClocOptions()

	paths := []string{
		dir,
	}

	processor := gocloc.NewProcessor(languages, options)

	result, err := processor.Analyze(paths)
	if err != nil {
		return -1, nil
	}

	selfWrittenGoCode, ok := result.Languages["Go"]
	if ok {
		return int(selfWrittenGoCode.Code), nil
	}

	return 0, fmt.Errorf("project does not have code written in Go")
}

// CalculateLibraryLOC is a function that calculates the library code lines within
// a repository. Basically the projects listed in the "go.mod" direct and indirect
// requirements.
func (r *Repo) CalculateLibraryLOC() (int, error) {
	// TODO
	modFilePath, err := utils.FindFile(r.dir, "go.mod")
	if err != nil {
		return -1, err
	}

	modFile := utils.NewModFile(modFilePath)
	err = modFile.Parse()
	if err != nil {
		return -1, err
	}

	// libraries are in format of golang.org/x/mod v0.15.0

	// libraryCount := 0

	goPath, err := exec.Command("go", "env", "GOPATH").Output()
	if err != nil {
		r.log.Errorf("unable to exec command: %s", err.Error())

		return 0, err
	}

	for _, lib := range modFile.Replace {
		l := strings.Split(lib, " ")
		name := l[0]
		version := l[1]

		before, _, ok := strings.Cut(string(goPath), "\n")
		if ok {
			libraryPath := before + "/" + "pkg" + "/" + "mod" + "/" + name + "@" + version

			// r.log.Debugf("go get -v %s@%s", name, version)
			r.log.Debugf("Library Path: %s", libraryPath)
		}
	}

	for _, lib := range modFile.Require {
		l := strings.Split(lib, " ")
		name := l[0]
		version := l[1]

		before, _, ok := strings.Cut(string(goPath), "\n")
		if ok {
			libraryPath := before + "/" + "pkg" + "/" + "mod" + "/" + name + "@" + version

			// r.log.Debugf("go get -v %s@%s", name, version)
			r.log.Debugf("Library Path: %s", libraryPath)
		}
	}

	// mkdir tmp
	// cd tmp && go mod init temp
	//

	// $(go env GOPATH)/pkg/mod

	// TODO
	// parse libraries from go.mod file
	// -> clone
	// -> calculate
	// -> append
	// -> delete

	return -1, nil
}

// func (r *Repo) CalculateService() {
// 	go func() {
// 		for lib := range r.tasks {
// 			fmt.Println(lib)
// 		}
// 	}()
// }
