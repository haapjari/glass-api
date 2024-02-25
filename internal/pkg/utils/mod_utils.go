package utils

import (
	"os"

	"golang.org/x/mod/modfile"
)

type ModFile struct {
	Path    string
	Module  string
	Require []string
	Replace []string
}

func NewModFile(path string) *ModFile {
	return &ModFile{
		Path:    path,
		Module:  "",
		Require: []string{},
		Replace: []string{},
	}
}

// Parses the "go.mod" file and populates required fields to the struct.
func (m *ModFile) Parse() error {
	bytes, err := os.ReadFile(m.Path)
	if err != nil {
		return err
	}

	file, err := modfile.Parse(m.Path, bytes, nil)
	if err != nil {
		return err
	}

	requirementsSlice := make([]string, len(file.Require))
	replacementsSlice := make([]string, len(file.Replace))

	for i := 0; i < len(file.Require); i++ {
		requirementsSlice[i] = file.Require[i].Mod.Path + " " + file.Require[i].Mod.Version
	}

	for i := 0; i < len(file.Replace); i++ {
		replacementsSlice[i] = file.Replace[i].New.Path
	}

	m.Require = requirementsSlice
	m.Replace = replacementsSlice

	return nil
}
