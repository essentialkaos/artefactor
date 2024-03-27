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
	"strconv"
	"time"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/fsutil"
	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/req"
	"github.com/essentialkaos/ek/v12/spinner"
	"github.com/essentialkaos/ek/v12/support"
	"github.com/essentialkaos/ek/v12/support/deps"
	"github.com/essentialkaos/ek/v12/terminal/tty"
	"github.com/essentialkaos/ek/v12/timeutil"
	"github.com/essentialkaos/ek/v12/tmp"
	"github.com/essentialkaos/ek/v12/usage"
	"github.com/essentialkaos/ek/v12/usage/completion/bash"
	"github.com/essentialkaos/ek/v12/usage/completion/fish"
	"github.com/essentialkaos/ek/v12/usage/completion/zsh"
	"github.com/essentialkaos/ek/v12/usage/man"
	"github.com/essentialkaos/ek/v12/usage/update"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Basic application info
const (
	APP  = "artefactor"
	VER  = "0.4.2"
	DESC = "Utility for downloading artefacts from GitHub"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Options
const (
	OPT_LIST     = "L:list"
	OPT_SOURCES  = "s:sources"
	OPT_NAME     = "n:name"
	OPT_TOKEN    = "t:token"
	OPT_UNIT     = "u:unit"
	OPT_NO_COLOR = "nc:no-color"
	OPT_HELP     = "h:help"
	OPT_VER      = "v:version"

	OPT_VERB_VER     = "vv:verbose-version"
	OPT_COMPLETION   = "completion"
	OPT_GENERATE_MAN = "generate-man"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// optMap contains information about all supported options
var optMap = options.Map{
	OPT_LIST:     {Type: options.BOOL},
	OPT_SOURCES:  {Value: "artefacts.yml"},
	OPT_NAME:     {},
	OPT_TOKEN:    {},
	OPT_UNIT:     {Type: options.BOOL},
	OPT_NO_COLOR: {Type: options.BOOL},
	OPT_HELP:     {Type: options.BOOL},
	OPT_VER:      {Type: options.MIXED},

	OPT_VERB_VER:     {Type: options.BOOL},
	OPT_COMPLETION:   {},
	OPT_GENERATE_MAN: {Type: options.BOOL},
}

var temp *tmp.Temp
var colorTagApp, colorTagVer string

// ////////////////////////////////////////////////////////////////////////////////// //

// Run is main utility function
func Run(gitRev string, gomod []byte) {
	preConfigureUI()

	args, errs := options.Parse(optMap)

	if len(errs) != 0 {
		printError(errs[0].Error())
		os.Exit(1)
	}

	configureUI()

	switch {
	case options.Has(OPT_COMPLETION):
		os.Exit(printCompletion())
	case options.Has(OPT_GENERATE_MAN):
		printMan()
		os.Exit(0)
	case options.GetB(OPT_VER):
		genAbout(gitRev).Print(options.GetS(OPT_VER))
		os.Exit(0)
	case options.GetB(OPT_VERB_VER):
		support.Collect(APP, VER).
			WithRevision(gitRev).
			WithDeps(deps.Extract(gomod)).
			WithChecks(checkGithubAvailability()...).
			Print()
		os.Exit(0)
	case options.GetB(OPT_HELP) || len(args) == 0:
		genUsage().Print()
		os.Exit(0)
	}

	err := prepare()

	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	dataDir := args.Get(0).Clean().String()

	if options.GetB(OPT_LIST) {
		listArtefacts(dataDir)
		return
	}

	err = process(dataDir)

	temp.Clean()

	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}
}

// preConfigureUI preconfigures UI based on information about user terminal
func preConfigureUI() {
	if !tty.IsTTY() {
		fmtc.DisableColors = true
	}

	switch {
	case fmtc.Is256ColorsSupported():
		colorTagApp, colorTagVer = "{*}{#117}", "{#117}"
	default:
		colorTagApp, colorTagVer = "{*}{c}", "{c}"
	}
}

// configureUI configures user interface
func configureUI() {
	if options.GetB(OPT_NO_COLOR) {
		fmtc.DisableColors = true
	}

	if options.GetB(OPT_UNIT) {
		fmtc.DisableColors = true
		spinner.DisableAnimation = true
	}

	fmtutil.SizeSeparator = " "
}

// prepare preconfigures app
func prepare() error {
	var err error

	temp, err = tmp.NewTemp()

	if err != nil {
		return err
	}

	req.SetUserAgent(APP, VER)

	return nil
}

// process starts arguments processing
func process(dataDir string) error {
	err := fsutil.ValidatePerms("DWRX", dataDir)

	if err != nil {
		return err
	}

	artefacts, err := parseArtefacts(options.GetS(OPT_SOURCES))

	if err != nil {
		return err
	}

	err = artefacts.Validate()

	if err != nil {
		return err
	}

	return downloadArtefacts(artefacts, dataDir)
}

// printError prints error message to console
func printError(f string, a ...interface{}) {
	if len(a) == 0 {
		fmtc.Fprintln(os.Stderr, "{r}▲ "+f+"{!}")
	} else {
		fmtc.Fprintf(os.Stderr, "{r}▲ "+f+"{!}\n", a...)
	}
}

// ////////////////////////////////////////////////////////////////////////////////// //

// checkGithubAvailability checks GitHub API availability
func checkGithubAvailability() []support.Check {
	var chks []support.Check

	headers := req.Headers{"X-GitHub-Api-Version": _API_VERSION}

	if options.Has(OPT_TOKEN) {
		headers["Authorization"] = "Bearer " + options.GetS(OPT_TOKEN)
	}

	resp, err := req.Request{
		URL:         "https://api.github.com/octocat",
		Headers:     headers,
		AutoDiscard: true,
	}.Get()

	if err != nil {
		chks = append(chks, support.Check{
			support.CHECK_ERROR, "GitHub API", "Can't send request",
		})
		return chks
	} else if resp.StatusCode != 200 {
		chks = append(chks, support.Check{
			support.CHECK_ERROR, "GitHub API", fmt.Sprintf(
				"API returned non-ok status code %s", resp.StatusCode,
			),
		})
		return chks
	}

	chks = append(chks, support.Check{
		support.CHECK_OK, "GitHub API", "API available",
	})

	remaining := resp.Header.Get("X-Ratelimit-Remaining")
	used := resp.Header.Get("X-Ratelimit-Used")
	limit := resp.Header.Get("X-Ratelimit-Limit")
	resetTS, _ := strconv.ParseInt(resp.Header.Get("X-Ratelimit-Reset"), 10, 64)
	resetDate := time.Unix(resetTS, 0)

	chk := support.Check{
		Title: "API Ratelimit",
		Message: fmt.Sprintf(
			"(Used: %s/%s | Reset: %s)",
			used, limit, timeutil.Format(resetDate, "%Y/%m/%d %H:%M:%S"),
		),
	}

	switch remaining {
	case "":
		chk.Status = support.CHECK_WARN
		chk.Message = "No info about ratelimit"
	case "0":
		chk.Status = support.CHECK_ERROR
		chk.Message = "No remaining requests " + chk.Message
	default:
		chk.Status = support.CHECK_OK
		chk.Message = "OK " + chk.Message
	}

	chks = append(chks, chk)

	return chks
}

// printCompletion prints completion for given shell
func printCompletion() int {
	info := genUsage()

	switch options.GetS(OPT_COMPLETION) {
	case "bash":
		fmt.Print(bash.Generate(info, "artefactor"))
	case "fish":
		fmt.Print(fish.Generate(info, "artefactor"))
	case "zsh":
		fmt.Print(zsh.Generate(info, optMap, "artefactor"))
	default:
		return 1
	}

	return 0
}

// printMan prints man page
func printMan() {
	fmt.Println(
		man.Generate(
			genUsage(),
			genAbout(""),
		),
	)
}

// genUsage generates usage info
func genUsage() *usage.Info {
	info := usage.NewInfo("", "data-dir")

	if fmtc.Is256ColorsSupported() {
		info.AppNameColorTag = colorTagApp
	}

	info.AddOption(OPT_LIST, "List downloaded artefacts in data directory")
	info.AddOption(OPT_SOURCES, "Path to YAML file with sources {s-}(default: artefacts.yml){!}", "file")
	info.AddOption(OPT_NAME, "Artefact name to download", "name")
	info.AddOption(OPT_TOKEN, "GitHub personal token", "token")
	info.AddOption(OPT_UNIT, "Run application in unit mode {s-}(no colors and animations){!}")
	info.AddOption(OPT_NO_COLOR, "Disable colors in output")
	info.AddOption(OPT_HELP, "Show this help message")
	info.AddOption(OPT_VER, "Show version")

	info.AddExample(
		"data",
		"Download artefacts to data directory",
	)

	info.AddExample(
		"-s ~/artefacts-all.yml data",
		"Download artefacts from given file to data directory",
	)

	info.AddExample(
		"-s ~/artefacts-all.yml -n shellcheck data",
		"Download shellcheck artefacts from given file to data directory",
	)

	return info
}

// genAbout generates info about version
func genAbout(gitRev string) *usage.About {
	about := &usage.About{
		App:           APP,
		Version:       VER,
		Desc:          DESC,
		Year:          2009,
		Owner:         "ESSENTIAL KAOS",
		License:       "Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>",
		UpdateChecker: usage.UpdateChecker{"essentialkaos/artefactor", update.GitHubChecker},
	}

	if gitRev != "" {
		about.Build = "git:" + gitRev
	}

	if fmtc.Is256ColorsSupported() {
		about.AppNameColorTag = colorTagApp
		about.VersionColorTag = colorTagVer
		about.DescSeparator = "{s}—{!}"
	}

	return about
}

// ////////////////////////////////////////////////////////////////////////////////// //
