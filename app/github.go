package app

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/req"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	_API_VERSION = "2022-11-28"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type ghRelease struct {
	Version     string     `json:"tag_name"`
	PublishDate time.Time  `json:"published_at"`
	Assets      []*ghAsset `json:"assets"`
}

type ghAsset struct {
	URL string `json:"browser_download_url"`
}

// ////////////////////////////////////////////////////////////////////////////////// //

// releaseCache is cache for github releases data
var releaseCache = map[string]*ghRelease{}

// ////////////////////////////////////////////////////////////////////////////////// //

// getLatestReleaseVersion returns the latest version of release
func getLatestReleaseVersion(repo string) (string, time.Time, error) {
	release, err := getLatestReleaseInfo(repo)

	if err != nil {
		return "", time.Time{}, err
	}

	return strings.TrimLeft(release.Version, "v"), release.PublishDate, nil
}

// getLatestReleaseAssets returns slice with URLs from the latest release
func getLatestReleaseAssets(repo string) ([]string, error) {
	release, err := getLatestReleaseInfo(repo)

	if err != nil {
		return nil, err
	}

	var urls []string

	for _, asset := range release.Assets {
		urls = append(urls, asset.URL)
	}

	return urls, nil
}

func getLatestReleaseInfo(repo string) (*ghRelease, error) {
	if releaseCache[repo] != nil {
		return releaseCache[repo], nil
	}

	headers := req.Headers{
		"X-GitHub-Api-Version": _API_VERSION,
	}

	if options.Has(OPT_TOKEN) {
		headers["Authorization"] = "Bearer " + options.GetS(OPT_TOKEN)
	}

	resp, err := req.Request{
		URL:         "https://api.github.com/repos/" + repo + "/releases/latest",
		Accept:      "application/vnd.github+json",
		Headers:     headers,
		AutoDiscard: true,
	}.Get()

	if err != nil {
		return nil, fmt.Errorf("Can't fetch GitHub data: %v", err)
	}

	release := &ghRelease{}
	err = resp.JSON(release)

	if err == nil {
		releaseCache[repo] = release
	}

	return release, err
}
