package app

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2025 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/fsutil"
	"github.com/essentialkaos/ek/v13/options"
	"github.com/essentialkaos/ek/v13/path"
	"github.com/essentialkaos/ek/v13/progress"
	"github.com/essentialkaos/ek/v13/req"
	"github.com/essentialkaos/ek/v13/system"
	"github.com/essentialkaos/ek/v13/terminal"

	"github.com/essentialkaos/artefactor/data"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// cmdGet is "get" command handler
func cmdGet(args options.Arguments) error {
	switch {
	case !args.Has(0):
		return fmt.Errorf("You must provide storage URL")
	case !args.Has(1):
		return fmt.Errorf("You must provide name of artefact")
	}

	storage := args.Get(0).String()
	index, err := readRemoteIndex(storage)

	if err != nil {
		return fmt.Errorf("Can't fetch index: %v", err)
	} else if index.IsEmpty() {
		return fmt.Errorf("Index is empty")
	}

	name := args.Get(1).String()
	artefactInfo := index.Find(name)

	if artefactInfo == nil {
		return fmt.Errorf("There is no artefact %q in storage", name)
	}

	version := args.Get(2).String()
	var versionInfo *data.ArtefactVersion

	if version == "" {
		versionInfo = artefactInfo.Latest()
	} else {
		versionInfo = artefactInfo.Find(version)
	}

	if versionInfo == nil {
		return fmt.Errorf("There is no version %s of %s", version, name)
	}

	return getArtefact(storage, name, versionInfo)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// getArtefact fetches all files of artefact
func getArtefact(storage, name string, info *data.ArtefactVersion) error {
	if !strings.HasPrefix(storage, "http") {
		storage = "https://" + storage
	}

	fmtc.Printfn("{*}Downloading files of {?primary}%s{!*}:%s{!}{*} artefact…{!}", name, info.Version)

	for _, file := range info.Files {
		fileName := stripArchFromBinaryName(file)
		fileURL := storage + "/" + path.Join(name, info.Version, file)

		err := fetchArtefactBinary(fileName, fileURL)

		if err != nil {
			terminal.Error("Error while downloading artefact binary: %v")
			fmtc.NewLine()
		}

		if options.GetB(OPT_INSTALL) {
			err = installArtefactBinary(fileName)

			if err != nil {
				terminal.Error("Error while installing artefact binary: %v")
				fmtc.NewLine()
			}
		}
	}

	return nil
}

// fetchArtefactBinary fetches artefact binary from remote storage
func fetchArtefactBinary(fileName, url string) error {
	pb := progress.New(0, fileName)

	pbs := progress.DefaultSettings
	pbs.BarFgColorTag = "{?primary}"
	pbs.NameColorTag = "{?primary}"
	pbs.PercentColorTag = "{*}"
	pbs.SpeedColorTag = "{s}"
	pbs.ProgressColorTag = "{s}"
	pbs.RemainingColorTag = "{s}"

	pb.UpdateSettings(pbs)

	resp, err := req.Request{
		URL:         url,
		AutoDiscard: true,
	}.Get()

	if err != nil {
		return err
	} else if resp.StatusCode != 200 {
		return fmt.Errorf("Storage returned non-ok status code %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	fd, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return fmt.Errorf("Can't create file %q: %v", fileName, err)
	}

	defer fd.Close()

	pb.SetTotal(resp.ContentLength)
	pb.Start()
	_, err = io.Copy(fd, pb.Reader(resp.Body))
	pb.Finish()

	if err != nil {
		return fmt.Errorf("Can't save binary: %v", err)
	}

	return nil
}

// installArtefactBinary installs binary to user binary directory ($HOME/.bin)
func installArtefactBinary(file string) error {
	if strings.Contains(file, ".") {
		terminal.Warn("File doesn't look like CLI, so we can't install it")
		return nil
	}

	user, err := system.CurrentUser()

	if err != nil {
		return fmt.Errorf("Can't get current user info: %v", err)
	}

	binDir := path.Join(user.HomeDir, ".bin")

	if !fsutil.CheckPerms("DRWX", binDir) {
		terminal.Warn(
			"There is no directory for user binaries (%s), so we can't install binary to it",
			binDir,
		)
		return nil
	}

	return fsutil.MoveFile(file, path.Join(binDir, file), 0755)
}

// stripArchFromBinaryName removes arch from file name
func stripArchFromBinaryName(file string) string {
	return strings.ReplaceAll(file, "-x86_64", "")
}
