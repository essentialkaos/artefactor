package data

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os"
	"strings"

	"github.com/essentialkaos/ek/v12/fsutil"
	"github.com/essentialkaos/ek/v12/jsonutil"
	"github.com/essentialkaos/ek/v12/path"
	"github.com/essentialkaos/ek/v12/sortutil"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type Index struct {
	Artefacts []*ArtefactInfo `json:"artefacts"`
}

// ArtefactInfo contains info about artefact
type ArtefactInfo struct {
	Name     string             `json:"name"`
	Versions []*ArtefactVersion `json:"versions"`
}

// ArtefactVersion contains info about artefact version
type ArtefactVersion struct {
	Files   []string `json:"files"`
	Version string   `json:"version"`
	Size    int64    `json:"size"`
}

// ////////////////////////////////////////////////////////////////////////////////// //

// BuildIndex builds index of artefacts
func BuildIndex(dir string) (*Index, error) {
	err := fsutil.ValidatePerms("DRX", dir)

	if err != nil {
		return nil, err
	}

	index := &Index{}
	dirs := fsutil.List(dir, true, fsutil.ListingFilter{Perms: "DRX"})

	if len(dirs) == 0 {
		return nil, fmt.Errorf("Data directory is empty")
	}

	sortutil.StringsNatural(dirs)

	for _, name := range dirs {
		versions := fsutil.List(path.Join(dir, name), true, fsutil.ListingFilter{
			NotMatchPatterns: []string{"latest"},
		})

		if len(versions) == 0 {
			continue
		}

		sortutil.Versions(versions)

		info := &ArtefactInfo{Name: name}
		index.Artefacts = append(index.Artefacts, info)

		for _, version := range versions {
			versionDir := path.Join(dir, name, version)

			files := fsutil.List(versionDir, false, fsutil.ListingFilter{Perms: "FR"})
			size := getVersionDataSize(versionDir, files)

			info.Versions = append(info.Versions, &ArtefactVersion{
				Version: version,
				Files:   files,
				Size:    size,
			})
		}
	}

	return index, nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// IsEmpty returns true if index is empty
func (i *Index) IsEmpty() bool {
	return i == nil || len(i.Artefacts) == 0
}

// Find searches for artefact with given name
func (i *Index) Find(name string) *ArtefactInfo {
	if i == nil {
		return nil
	}

	name = strings.ToLower(name)

	for _, a := range i.Artefacts {
		if strings.ToLower(a.Name) == name {
			return a
		}
	}

	return nil
}

// Write writes index data into the file
func (i *Index) Write(file string) error {
	if i == nil {
		return fmt.Errorf("Index is nil")
	}

	err := jsonutil.Write(file, i, 0644)

	if err != nil {
		return err
	}

	return os.Chmod(file, 0644)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Find searches info about given version
func (i *ArtefactInfo) Find(version string) *ArtefactVersion {
	if i == nil || len(i.Versions) == 0 {
		return nil
	}

	for _, v := range i.Versions {
		if v.Version == version {
			return v
		}
	}

	return nil
}

// Latest returns info about the latest version of artefact
func (i *ArtefactInfo) Latest() *ArtefactVersion {
	if i == nil || len(i.Versions) == 0 {
		return nil
	}

	return i.Versions[len(i.Versions)-1]
}

// ////////////////////////////////////////////////////////////////////////////////// //

// getVersionDataSize returns size of all version files
func getVersionDataSize(versionDir string, files []string) int64 {
	var result int64

	for _, file := range files {
		result += fsutil.GetSize(path.Join(versionDir, file))
	}

	return result
}
