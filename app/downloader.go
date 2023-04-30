package app

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/fsutil"
	"github.com/essentialkaos/ek/v12/httputil"
	"github.com/essentialkaos/ek/v12/path"
	"github.com/essentialkaos/ek/v12/req"
	"github.com/essentialkaos/ek/v12/spinner"
	"github.com/essentialkaos/ek/v12/strutil"
	"github.com/essentialkaos/ek/v12/timeutil"

	"github.com/essentialkaos/npck"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// downloadArtefacts downloads artefacts from GitHub if required
func downloadArtefacts(artefacts Artefacts, dataDir string) error {
	var isFailed bool

	fmtc.NewLine()

	for _, artefact := range artefacts {
		err := downloadArtefact(artefact, dataDir)

		if err != nil {
			fmtc.Printf("   {r}%v{!}\n", err)
			isFailed = true
		}

		temp.Clean()
		fmtc.NewLine()
	}

	if isFailed {
		return fmt.Errorf("Some artefacts can not be downloaded from GitHub")
	}

	return nil
}

func downloadArtefact(artefact *Artefact, dataDir string) error {
	fmtc.Printf(
		"{*}Downloading {c}%s{!}{*} from {s}%s{!}{*}…{!}\n",
		artefact.Name, artefact.Repo,
	)

	spinner.Show("Checking the latest version on GitHub")
	version, pubDate, err := getLatestReleaseVersion(artefact.Repo)
	spinner.Done(err == nil)

	if err != nil {
		return err
	}

	fmtc.Printf(
		"   Found version: {g}%s{!} {s-}(%s){!}\n",
		version, timeutil.Format(pubDate, "%Y/%m/%d %H:%M"),
	)

	releaseDir := path.Join(dataDir, strutil.Q(artefact.Dir, artefact.Name), version)
	latestLink := path.Join(dataDir, strutil.Q(artefact.Dir, artefact.Name), "latest")
	outputFile := path.Join(releaseDir, artefact.Output)

	if fsutil.IsExist(outputFile) {
		fmtc.Println("   {g}The latest version already downloaded{!}")
		return nil
	}

	err = downloadArtefactData(artefact, version, releaseDir, outputFile)

	if err != nil {
		return err
	}

	if fsutil.IsExist(latestLink) {
		os.Remove(latestLink)
	}

	err = os.Symlink(releaseDir, latestLink)

	if err != nil {
		return fmt.Errorf("Can't create link to the latest release: %v", err)
	}

	binarySize := fsutil.GetSize(outputFile)

	fmtc.Printf(
		"   {g}Artefact successfully downloaded {s}%s{g} and saved to data directory{!}\n",
		fmtutil.PrettySize(binarySize),
	)

	return nil
}

// downloadArtefactData downloads and stores artefact
func downloadArtefactData(artefact *Artefact, version, outputDir, outputFile string) error {
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
func downloadArtefactFile(artefact *Artefact, version string) (string, error) {
	url, err := getArtefactBinaryURL(artefact, version)

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

	w := bufio.NewWriter(tempFd)
	_, err = io.Copy(w, resp.Response.Body)

	if err != nil {
		return "", err
	}

	w.Flush()

	return tempName, nil
}

// unpackArtefactArchive
func unpackArtefactArchive(artefact *Artefact, file string) (string, error) {
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
func getArtefactBinaryURL(artefact *Artefact, version string) (string, error) {
	if httputil.IsURL(artefact.Source) {
		return strings.ReplaceAll(artefact.Source, "{version}", version), nil
	}

	assets, err := getLatestReleaseAssets(artefact.Repo)

	if err != nil {
		return "", err
	}

	for _, url := range assets {
		file := path.Base(url)
		match, _ := path.Match(artefact.Source, file)

		if match {
			return url, nil
		}
	}

	return "", fmt.Errorf("Can't find binary URL")
}

// getArtefactExt returns extension for artefact file
func getArtefactExt(artefact *Artefact) string {
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
func isArchive(artefact *Artefact) bool {
	return getArtefactExt(artefact) != ""
}
