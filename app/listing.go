package app

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/fsutil"
	"github.com/essentialkaos/ek/v12/pager"
	"github.com/essentialkaos/ek/v12/path"
	"github.com/essentialkaos/ek/v12/sortutil"
	"github.com/essentialkaos/ek/v12/terminal/tty"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type ArtefactInfo struct {
	Name     string
	Versions []*ArtefactVersion
}

type ArtefactVersion struct {
	Version string
	Size    int64
}

// ////////////////////////////////////////////////////////////////////////////////// //

// listArtefacts prints list of downloaded artefacts
func listArtefacts(dataDir string) {
	artefacts := getArtefacts(dataDir)

	if len(artefacts) == 0 {
		fmtc.Println("{y}No artefacts found{!}")
		return
	}

	if tty.IsTTY() {
		if pager.Setup() == nil {
			defer pager.Complete()
		}
	}

	for _, info := range artefacts {
		size := fmtutil.PrettyNum(len(info.Versions))

		fmtc.Printf("{s-}┌{!}{*@} %s {!}{#240}{*@} %s {!}\n", info.Name, size)
		fmtc.Printf("{s-}│{!}\n")

		for i, version := range info.Versions {
			if i+1 != len(info.Versions) {
				fmtc.Printf(
					"{s-}├{!} {s}%s{!} {s-}(%s){!}\n",
					version.Version,
					fmtutil.PrettySize(version.Size),
				)
			} else {
				fmtc.Printf(
					"{s-}└{!} {*}%s{!} {s-}(%s){!}\n",
					version.Version,
					fmtutil.PrettySize(version.Size),
				)
			}
		}

		fmtc.NewLine()
	}
}

// getArtefacts returns info about all artefacts in given directory
func getArtefacts(dataDir string) []*ArtefactInfo {
	dirs := fsutil.List(dataDir, true, fsutil.ListingFilter{Perms: "DRX"})

	if len(dirs) == 0 {
		return nil
	}

	var result []*ArtefactInfo

	sortutil.StringsNatural(dirs)

	for _, name := range dirs {
		versions := fsutil.List(path.Join(dataDir, name), true, fsutil.ListingFilter{
			NotMatchPatterns: []string{"latest"},
		})

		if len(versions) == 0 {
			continue
		}

		info := &ArtefactInfo{Name: name}
		result = append(result, info)

		for _, version := range versions {
			dataSize := getVersionDataSize(path.Join(dataDir, name, version))
			info.Versions = append(info.Versions, &ArtefactVersion{version, dataSize})
		}
	}

	return result
}

// getVersionDataSize returns size of all version files
func getVersionDataSize(dir string) int64 {
	var result int64

	files := fsutil.List(dir, false, fsutil.ListingFilter{Perms: "FR"})

	for _, file := range files {
		result += fsutil.GetSize(path.Join(dir, file))
	}

	return result
}
