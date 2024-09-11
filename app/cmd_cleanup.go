package app

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/mathutil"
	"github.com/essentialkaos/ek/v13/options"
	"github.com/essentialkaos/ek/v13/spinner"
	"github.com/essentialkaos/ek/v13/terminal"
	"github.com/essentialkaos/ek/v13/terminal/input"

	"github.com/essentialkaos/artefactor/data"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// MIN_VERSIONS is minimum versions to keep
const MIN_VERSIONS = 5

// ////////////////////////////////////////////////////////////////////////////////// //

// cmdCleanup is "cleanup" command handler
func cmdCleanup(args options.Arguments) error {
	if !args.Has(0) {
		return fmt.Errorf("You must provide path to data directory")
	}

	var err error
	var keepVersions int

	if args.Has(1) {
		keepVersions, err = args.Get(1).Int()

		if err != nil {
			return fmt.Errorf("Can't parse minimum number of versions: %v", err)
		}
	}

	keepVersions = mathutil.Max(keepVersions, MIN_VERSIONS)

	dataDir := args.Get(0).Clean().String()
	index, err := readLocalIndex(dataDir)

	if err != nil {
		return fmt.Errorf("Can't get index data: %v", err)
	} else if index.IsEmpty() {
		terminal.Warn("No artefacts found")
		return nil
	}

	ok, _ := input.ReadAnswer(
		fmt.Sprintf("Remove old versions except the last %d?", keepVersions), "N",
	)

	if !ok {
		return nil
	}

	fmtc.NewLine()

	err = cleanupVersions(index, dataDir, keepVersions)

	if err != nil {
		return err
	}

	return rebuildIndex(dataDir)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// cleanupVersions removes outdated versions
func cleanupVersions(index *data.Index, dataDir string, keepVersions int) error {
	var versionNum int

	allVersions := getVersionToRemove(index, dataDir, keepVersions)

	if len(allVersions) == 0 {
		terminal.Warn("No versions to clean")
		return nil
	}

	for name, versions := range allVersions {
		fmtc.Printf(" {s-}-{!} {*}%s{!}{s}:{!} ", name)
		fmtc.Print(strings.Join(versions, "{s},{!} "))
		fmtc.NewLine()

		versionNum += len(versions)
	}

	fmtc.NewLine()

	ok, _ := input.ReadAnswer(
		fmt.Sprintf("Remove these versions (%d)?", versionNum), "N",
	)

	if !ok {
		return nil
	}

	fmtc.NewLine()

	var currentVersion int

	spinner.Show("Removing outdated versions")

	for name, versions := range allVersions {
		for _, v := range versions {
			versionPath := path.Join(dataDir, name, v)

			currentVersion++

			spinner.Update(
				"{s}[%d/%d]{!} Removing {?primary}%s{!*}:%s{!}",
				currentVersion, versionNum, name, v,
			)

			os.RemoveAll(versionPath)
		}
	}

	spinner.Update("{s}[%d/%d]{!} Remove outdated versions", currentVersion, versionNum)
	spinner.Done(true)

	return nil
}

// getVersionToRemove returns map with info about outdated versions
func getVersionToRemove(index *data.Index, dataDir string, keepVersions int) map[string][]string {
	versions := map[string][]string{}

	for _, a := range index.Artefacts {
		if len(a.Versions) < keepVersions {
			continue
		}

		for i := 0; i < len(a.Versions)-keepVersions; i++ {
			versions[a.Name] = append(versions[a.Name], a.Versions[i].Version)
		}
	}

	return versions
}
