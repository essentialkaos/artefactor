package app

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
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

// listArtefacts prints list of downloaded artefacts
func listArtefacts(dataDir string) {
	dirs := fsutil.List(dataDir, true, fsutil.ListingFilter{Perms: "DRX"})

	if len(dirs) == 0 {
		fmtc.Println("{y}No artefacts found{!}")
		return
	}

	if tty.IsTTY() {
		if pager.Setup() == nil {
			defer pager.Complete()
		}
	}

	sortutil.StringsNatural(dirs)
	fmtc.NewLine()

	for _, name := range dirs {
		listArtefactVersions(name, path.Join(dataDir, name))
		fmtc.NewLine()
	}
}

// listArtefactVersions prints list of all versions
func listArtefactVersions(name, dir string) {
	versions := fsutil.List(dir, true, fsutil.ListingFilter{
		NotMatchPatterns: []string{"latest"},
	})

	if len(versions) == 0 {
		fmtc.Printf("{s-}%s{!}\n", name)
		return
	}

	fmtc.Printf("{s-}┌{!}{*@} %s {!}{#240}{@} %s {!}\n", name, fmtutil.PrettyNum(len(versions)))
	fmtc.Printf("{s-}│{!}\n")

	sortutil.Versions(versions)

	for i, version := range versions {
		dataSize := getVersionDataSize(path.Join(dir, version))

		if i+1 != len(versions) {
			fmtc.Printf("{s-}├{!} %s {s-}(%s){!}\n", version, fmtutil.PrettySize(dataSize))
		} else {
			fmtc.Printf("{s-}└{!} %s {s-}(%s){!}\n", version, fmtutil.PrettySize(dataSize))
		}
	}
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
