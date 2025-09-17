package app

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2025 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/fmtutil"
	"github.com/essentialkaos/ek/v13/fsutil"
	"github.com/essentialkaos/ek/v13/httputil"
	"github.com/essentialkaos/ek/v13/options"
	"github.com/essentialkaos/ek/v13/path"
	"github.com/essentialkaos/ek/v13/req"
	"github.com/essentialkaos/ek/v13/spinner"
	"github.com/essentialkaos/ek/v13/strutil"
	"github.com/essentialkaos/ek/v13/timeutil"

	"github.com/essentialkaos/npck"

	"github.com/essentialkaos/artefactor/data"
	"github.com/essentialkaos/artefactor/github"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// cmdDownload is "download" command handler
func cmdDownload(args options.Arguments) error {
	if !args.Has(0) {
		return fmt.Errorf("You must provide path to data directory")
	}

	dataDir := args.Get(0).Clean().String()
	artefactName := args.Get(1).String()

	err := fsutil.ValidatePerms("DWRX", dataDir)

	if err != nil {
		return err
	}

	artefacts, err := data.ReadArtefacts(options.GetS(OPT_SOURCES))

	if err != nil {
		return err
	}

	err = artefacts.Validate()

	if err != nil {
		return err
	}

	return downloadArtefacts(artefacts, dataDir, artefactName)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// downloadArtefacts downloads artefacts from GitHub if required
func downloadArtefacts(artefacts data.Artefacts, dataDir, artefactName string) error {
	var isFailed bool

	fmtc.NewLine()

	for _, artefact := range artefacts {
		if artefactName != "" && artefactName != artefact.Name {
			continue
		}

		err := downloadArtefact(artefact, dataDir)

		if err != nil {
			fmtc.Printfn("   {r}%v{!}", err)
			isFailed = true
		}

		temp.Clean()
		fmtc.NewLine()
	}

	restorePermissions(dataDir)

	if isFailed {
		return fmt.Errorf("Some artefacts can not be downloaded from GitHub")
	}

	return rebuildIndex(dataDir)
}

// downloadArtefact downloads specified artefact
func downloadArtefact(artefact *data.Artefact, dataDir string) error {
	fmtc.Printfn(
		"{*}Downloading {c}%s{!}{*} from {s}%s{!}{*}…{!}",
		artefact.Name, artefact.Repo,
	)

	spinner.Show("Checking the latest version on GitHub")
	version, pubDate, err := github.GetLatestReleaseVersion(artefact.Repo)
	spinner.Done(err == nil)

	if err != nil {
		return err
	}

	fmtc.Printfn(
		"   Found version: {g}%s{!} {s-}(%s){!}",
		version, timeutil.Format(pubDate, "%Y/%m/%d %H:%M"),
	)

	artefact.ApplyVersion(version)

	releaseDir := path.Join(dataDir, strutil.Q(artefact.Dir, artefact.Name), version)
	outputFile := path.Join(releaseDir, artefact.Output)

	if fsutil.IsExist(outputFile) {
		modDate, err := fsutil.GetMTime(outputFile)

		if err == nil && modDate.After(pubDate) {
			fmtc.Println("   {s}There is no update available for this application{!}")
			return nil
		}
	}

	err = downloadArtefactData(artefact, version, releaseDir, outputFile)

	if err != nil {
		return err
	}

	latestLink := path.Join(dataDir, strutil.Q(artefact.Dir, artefact.Name), "latest")

	if fsutil.IsLink(latestLink) {
		os.Remove(latestLink)
	}

	if !fsutil.IsExist(latestLink) {
		err = os.Symlink(version, latestLink)

		if err != nil {
			return fmt.Errorf("Can't create link to the latest release: %v", err)
		}
	}

	binarySize := fsutil.GetSize(outputFile)

	fmtc.Printfn(
		"   {g}Artefact successfully downloaded (%s) and saved to data directory{!}",
		fmtutil.PrettySize(binarySize),
	)

	return nil
}

// downloadArtefactData downloads and stores artefact
func downloadArtefactData(artefact *data.Artefact, version, outputDir, outputFile string) error {
	spinner.Show("Downloading binary from GitHub")
	binFile, err := downloadArtefactFile(artefact, version)
	spinner.Done(err == nil)

	if err != nil {
		return err
	}

	if isArchive(artefact) {
		binFile, err = unpackArtefactArchive(artefact, binFile)

		if err != nil {
			return err
		}
	}

	if !fsutil.IsExist(outputDir) {
		err = os.MkdirAll(outputDir, 0755)

		if err != nil {
			return err
		}
	}

	err = fsutil.CopyFile(binFile, outputFile, 0644)

	if err != nil {
		return err
	}

	return nil
}

// downloadArtefactFile downloads binary file
func downloadArtefactFile(artefact *data.Artefact, version string) (string, error) {
	url, err := getArtefactBinaryURL(artefact)

	if err != nil {
		return "", err
	}

	tempFd, tempName, err := temp.MkFile(artefact.Name + getArtefactExt(artefact))

	if err != nil {
		return "", err
	}

	resp, err := req.Request{
		URL:         url,
		AutoDiscard: true,
	}.Get()

	if err != nil {
		return "", err
	}

	w := bufio.NewWriter(tempFd)
	_, err = io.Copy(w, resp.Body)

	if err != nil {
		return "", err
	}

	w.Flush()

	return tempName, nil
}

// unpackArtefactArchive unpacks artefact from archive
func unpackArtefactArchive(artefact *data.Artefact, file string) (string, error) {
	spinner.Show("Unpacking data")

	tmpDir, err := temp.MkDir()

	if err != nil {
		spinner.Done(false)
		return "", err
	}

	err = npck.Unpack(file, tmpDir)

	if err != nil {
		spinner.Done(false)
		return "", fmt.Errorf("Can't unpack data: %v", err)
	}

	spinner.Done(true)

	if fsutil.CheckPerms("FRS", path.Join(tmpDir, artefact.File)) {
		return path.Join(tmpDir, artefact.File), nil
	}

	for _, file := range fsutil.ListAllFiles(tmpDir, true) {
		isMatch, _ := path.Match(artefact.File, file)

		if isMatch {
			return path.Join(tmpDir, file), nil
		}
	}

	return "", fmt.Errorf("Can't find binary \"%s\" in unpacked data", artefact.File)
}

// getArtefactBinaryURL returns URL of binary file
func getArtefactBinaryURL(artefact *data.Artefact) (string, error) {
	if httputil.IsURL(artefact.Source) {
		return artefact.Source, nil
	}

	assets, err := github.GetLatestReleaseAssets(artefact.Repo)

	if err != nil {
		return "", err
	}

	for _, url := range assets {
		file := path.Base(url)
		match, _ := path.Match(
			strings.ToLower(artefact.Source),
			strings.ToLower(file),
		)

		if match {
			return url, nil
		}
	}

	return "", fmt.Errorf("Can't find binary URL")
}

// getArtefactExt returns extension for artefact file
func getArtefactExt(artefact *data.Artefact) string {
	switch {
	case strings.HasSuffix(artefact.Source, ".tar.gz"),
		strings.HasSuffix(artefact.Source, ".tgz"):
		return ".tar.gz"
	case strings.HasSuffix(artefact.Source, ".tar.bz2"),
		strings.HasSuffix(artefact.Source, ".tbz2"):
		return ".tar.bz2"
	case strings.HasSuffix(artefact.Source, ".tar.xz"),
		strings.HasSuffix(artefact.Source, ".txz"):
		return ".tar.xz"
	case strings.HasSuffix(artefact.Source, ".zip"):
		return ".zip"
	}

	return ""
}

// isArchive returns true if given file is an archive
func isArchive(artefact *data.Artefact) bool {
	return getArtefactExt(artefact) != ""
}

// restorePermissions restores permissions for files and directories
func restorePermissions(dataDir string) {
	dirs := fsutil.ListAllDirs(dataDir, false)
	fsutil.ListToAbsolute(dataDir, dirs)

	for _, dir := range dirs {
		os.Chmod(dir, 0755)
	}

	files := fsutil.ListAllFiles(dataDir, false)
	fsutil.ListToAbsolute(dataDir, files)

	for _, file := range files {
		os.Chmod(file, 0644)
	}
}

// rebuildIndex rebuilds index
func rebuildIndex(dataDir string) error {
	index, err := data.BuildIndex(dataDir)

	if err != nil {
		return fmt.Errorf("Can't build index: %v", err)
	}

	err = index.Write(path.Join(dataDir, "index.json"))

	if err != nil {
		return fmt.Errorf("Can't save index: %v", err)
	}

	return nil
}
