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

	"github.com/essentialkaos/ek/v12/httputil"
	"github.com/essentialkaos/go-simpleyaml/v2"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Artefact contains info about artefact
type Artefact struct {
	Name   string
	Repo   string
	Output string
	Glob   string
	Binary string
	URL    string

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
			Glob:   info.Get("glob").MustString(""),
			Binary: info.Get("binary").MustString(""),
			URL:    info.Get("url").MustString(""),

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
	case a.Output == "":
		return fmt.Errorf("Artefact \"%s\" invalid: output can't be empty", a.Name)
	case a.Repo != "" && strings.Index(a.Repo, "/") == -1:
		return fmt.Errorf("Artefact \"%s\" invalid: repo name is invalid", a.Name)
	case a.URL != "" && !httputil.IsURL(a.URL):
		return fmt.Errorf("Artefact \"%s\" invalid: URL contains invalid value", a.Name)
	case a.URL != "" && a.Binary == "" && strings.HasSuffix(a.URL, ".tar.gz"),
		a.URL != "" && a.Binary == "" && strings.HasSuffix(a.URL, ".tar.xz"),
		a.URL != "" && a.Binary == "" && strings.HasSuffix(a.URL, ".zip"),
		a.URL != "" && a.Binary == "" && strings.HasSuffix(a.Glob, ".tar.gz"),
		a.URL != "" && a.Binary == "" && strings.HasSuffix(a.Glob, ".tar.xz"),
		a.URL != "" && a.Binary == "" && strings.HasSuffix(a.Glob, ".zip"):
		return fmt.Errorf("Artefact \"%s\" invalid: binary name is not defined for archive file", a.Name)
	}

	return nil
}
