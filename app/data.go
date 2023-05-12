package app

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os"
	"strings"

	"github.com/essentialkaos/go-simpleyaml/v2"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Artefact contains info about artefact
type Artefact struct {
	Name   string
	Repo   string
	Output string
	Source string
	File   string
	Dir    string

	index int
}

// Artefacts is a slice with artefacts
type Artefacts []*Artefact

// ////////////////////////////////////////////////////////////////////////////////// //

// ParseArtefacts parses YAML file with artefacts
func parseArtefacts(file string) (Artefacts, error) {
	yamlData, err := os.ReadFile(file)

	if err != nil {
		return nil, fmt.Errorf("Error while reading artefacts data: %v", err)
	}

	yaml, err := simpleyaml.NewYaml(yamlData)

	if err != nil {
		return nil, fmt.Errorf("Error while parsing artefacts data: %v", err)
	}

	return convertArtefactsYaml(yaml)
}

// convertArtefactsYaml converts yaml data into internal struct
func convertArtefactsYaml(yaml *simpleyaml.Yaml) (Artefacts, error) {
	if !yaml.IsArray() {
		return nil, fmt.Errorf("Wrong YAML format (must be array)")
	}

	var index int
	var result Artefacts

	for yaml.IsIndexExist(index) {
		info := yaml.GetByIndex(index)

		result = append(result, &Artefact{
			Name:   info.Get("name").MustString(""),
			Repo:   info.Get("repo").MustString(""),
			Output: info.Get("output").MustString(""),
			Source: info.Get("source").MustString(""),
			File:   info.Get("file").MustString(""),
			Dir:    info.Get("dir").MustString(""),

			index: index,
		})

		index++
	}

	return result, nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Validate validates all artefacts
func (a Artefacts) Validate() error {
	for _, artefact := range a {
		err := artefact.Validate()

		if err != nil {
			return err
		}
	}

	return nil
}

// Validate validates artefact info
func (a *Artefact) Validate() error {
	switch {
	case a.Name == "":
		return fmt.Errorf("Artefact %d invalid: name can't be empty", a.index)
	case a.Repo == "":
		return fmt.Errorf("Artefact \"%s\" invalid: repo can't be empty", a.Name)
	case a.Source == "":
		return fmt.Errorf("Artefact \"%s\" invalid: source can't be empty", a.Name)
	case a.Output == "":
		return fmt.Errorf("Artefact \"%s\" invalid: output can't be empty", a.Name)
	case a.Dir != "" && strings.Index(a.Dir, "/") != -1:
		return fmt.Errorf("Artefact \"%s\" invalid: dir must not contains /", a.Name)
	case a.Repo != "" && strings.Index(a.Repo, "/") == -1:
		return fmt.Errorf("Artefact \"%s\" invalid: repo name is invalid", a.Name)
	case a.File == "" && strings.HasSuffix(a.Source, ".tar.gz"),
		a.File == "" && strings.HasSuffix(a.Source, ".tar.xz"),
		a.File == "" && strings.HasSuffix(a.Source, ".zip"):
		return fmt.Errorf("Artefact \"%s\" invalid: file is not defined for archive file", a.Name)
	}

	return nil
}

// ApplyVersion applies version data to artefact
func (a *Artefact) ApplyVersion(version string) {
	if strings.Contains(a.File, "{version}") {
		a.File = strings.ReplaceAll(a.File, "{version}", version)
	}

	if strings.Contains(a.Source, "{version}") {
		a.Source = strings.ReplaceAll(a.Source, "{version}", version)
	}
}
