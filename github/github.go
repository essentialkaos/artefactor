package github

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2025 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v13/req"
	"github.com/essentialkaos/ek/v13/timeutil"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const API_VERSION = "2022-11-28"

// ////////////////////////////////////////////////////////////////////////////////// //

// Release contains info about release
type Release struct {
	Version     string    `json:"tag_name"`
	PublishDate time.Time `json:"published_at"`
	Assets      []*Asset  `json:"assets"`
}

// Asset contains info about release asset
type Asset struct {
	URL string `json:"browser_download_url"`
}

// Limits contains info about GitHubv API limits
type Limits struct {
	Used  int
	Total int
	Reset time.Time
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Token is GitHub access token
var Token string

// ////////////////////////////////////////////////////////////////////////////////// //

// cache is cache for github releases data
var cache = map[string]*Release{}

// ////////////////////////////////////////////////////////////////////////////////// //

// GetLimits returns info about limits
func GetLimits() (Limits, error) {
	headers := req.Headers{"X-GitHub-Api-Version": API_VERSION}

	if Token != "" {
		headers["Authorization"] = "Bearer " + Token
	}

	resp, err := req.Request{
		URL:         "https://api.github.com/octocat",
		Headers:     headers,
		AutoDiscard: true,
	}.Get()

	if err != nil {
		return Limits{}, fmt.Errorf("Can't send request")
	} else if resp.StatusCode != 200 {
		return Limits{}, fmt.Errorf("API returned non-ok status code %d", resp.StatusCode)
	}

	used, _ := strconv.Atoi(resp.Header.Get("X-Ratelimit-Used"))
	total, _ := strconv.Atoi(resp.Header.Get("X-Ratelimit-Limit"))
	resetTS, _ := strconv.ParseInt(resp.Header.Get("X-Ratelimit-Reset"), 10, 64)
	resetDate := time.Unix(resetTS, 0)

	return Limits{
		Used:  used,
		Total: total,
		Reset: resetDate,
	}, nil
}

// GetLatestReleaseVersion returns the latest version of release
func GetLatestReleaseVersion(repo string) (string, time.Time, error) {
	release, err := GetLatestReleaseInfo(repo)

	if err != nil {
		return "", time.Time{}, err
	}

	return strings.TrimLeft(release.Version, "v"), release.PublishDate, nil
}

// GetLatestReleaseAssets returns slice with URLs from the latest release
func GetLatestReleaseAssets(repo string) ([]string, error) {
	release, err := GetLatestReleaseInfo(repo)

	if err != nil {
		return nil, err
	}

	var urls []string

	for _, asset := range release.Assets {
		urls = append(urls, asset.URL)
	}

	return urls, nil
}

// GetLatestReleaseInfo returns info about the latest release
func GetLatestReleaseInfo(repo string) (*Release, error) {
	if cache[repo] != nil {
		return cache[repo], nil
	}

	headers := req.Headers{"X-GitHub-Api-Version": API_VERSION}

	if Token != "" {
		headers["Authorization"] = "Bearer " + Token
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

	if resp.Header.Get("X-Ratelimit-Remaining") == "0" {
		resetTS, _ := strconv.ParseInt(resp.Header.Get("X-Ratelimit-Reset"), 10, 64)
		resetDate := time.Unix(resetTS, 0)

		return nil, fmt.Errorf(
			"Reached limit for requests to GitHub API (%s/%s | %s to reset)",
			resp.Header.Get("X-Ratelimit-Used"),
			resp.Header.Get("X-Ratelimit-Limit"),
			timeutil.PrettyDuration(time.Until(resetDate)),
		)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub returned non-OK response code %d", resp.StatusCode)
	}

	release := &Release{}
	err = resp.JSON(release)

	if err == nil {
		cache[repo] = release
	} else {
		return nil, fmt.Errorf("Can't decode response JSON: %v", err)
	}

	return release, nil
}
