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

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/req"
	"github.com/essentialkaos/ek/v12/spinner"
	"github.com/essentialkaos/ek/v12/strutil"
	"github.com/essentialkaos/ek/v12/support"
	"github.com/essentialkaos/ek/v12/support/deps"
	"github.com/essentialkaos/ek/v12/terminal"
	"github.com/essentialkaos/ek/v12/terminal/input"
	"github.com/essentialkaos/ek/v12/terminal/tty"
	"github.com/essentialkaos/ek/v12/timeutil"
	"github.com/essentialkaos/ek/v12/tmp"
	"github.com/essentialkaos/ek/v12/usage"
	"github.com/essentialkaos/ek/v12/usage/completion/bash"
	"github.com/essentialkaos/ek/v12/usage/completion/fish"
	"github.com/essentialkaos/ek/v12/usage/completion/zsh"
	"github.com/essentialkaos/ek/v12/usage/man"
	"github.com/essentialkaos/ek/v12/usage/update"

	"github.com/essentialkaos/artefactor/github"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Basic application info
const (
	APP  = "artefactor"
	VER  = "0.5.0"
	DESC = "Utility for downloading artefacts from GitHub"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Options
const (
	OPT_SOURCES  = "s:sources"
	OPT_NAME     = "n:name"
	OPT_TOKEN    = "t:token"
	OPT_INSTALL  = "I:install"
	OPT_UNIT     = "u:unit"
	OPT_NO_COLOR = "nc:no-color"
	OPT_HELP     = "h:help"
	OPT_VER      = "v:version"

	OPT_VERB_VER     = "vv:verbose-version"
	OPT_COMPLETION   = "completion"
	OPT_GENERATE_MAN = "generate-man"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	CMD_DOWNLOAD = "download"
	CMD_GET      = "get"
	CMD_LIST     = "list"
	CMD_CLEANUP  = "cleanup"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// optMap contains information about all supported options
var optMap = options.Map{
	OPT_SOURCES:  {Value: "artefacts.yml"},
	OPT_TOKEN:    {},
	OPT_INSTALL:  {Type: options.BOOL},
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

	if !errs.IsEmpty() {
		terminal.Error("Options parsing errors:")
		terminal.Error(errs.String())
		os.Exit(1)
	}

	configureUI()

	github.Token = strutil.Q(options.GetS(OPT_TOKEN), os.Getenv("GITHUB_TOKEN"))

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
		terminal.Error(err.Error())
		os.Exit(1)
	}

	err = execCommand(args)

	if err != nil {
		terminal.Error(err.Error())
		temp.Clean()
		os.Exit(1)
	}
}

// preConfigureUI preconfigures UI based on information about user terminal
func preConfigureUI() {
	if !tty.IsTTY() {
		fmtc.DisableColors = true
	}

	input.TitleColorTag = "{s}{&}"
	input.Prompt = "{s}›{!} "
	input.MaskSymbol = "•"
	input.MaskSymbolColorTag = "{s-}"

	switch {
	case fmtc.Is256ColorsSupported():
		colorTagApp, colorTagVer = "{*}{#117}", "{#117}"
		fmtc.NameColor("primary", "{#117}")
	default:
		colorTagApp, colorTagVer = "{*}{c}", "{c}"
		fmtc.NameColor("primary", "{c}")
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

// execCommand executes command
func execCommand(args options.Arguments) error {
	var err error

	cmd := args.Get(0).String()
	args = args[1:]

	switch cmd {
	case CMD_DOWNLOAD:
		err = cmdDownload(args)
	case CMD_GET:
		err = cmdGet(args)
	case CMD_LIST:
		err = cmdList(args)
	case CMD_CLEANUP:
		err = cmdCleanup(args)
	default:
		return fmt.Errorf("Unknown command %q", cmd)
	}

	return err
}

// ////////////////////////////////////////////////////////////////////////////////// //

// checkGithubAvailability checks GitHub API availability
func checkGithubAvailability() []support.Check {
	var chks []support.Check

	limits, err := github.GetLimits()

	if err != nil {
		chks = append(chks, support.Check{
			support.CHECK_ERROR, "GitHub API", err.Error(),
		})

		return chks
	}

	chks = append(chks, support.Check{
		support.CHECK_OK, "GitHub API", "API available",
	})

	chk := support.Check{
		Title: "API Ratelimit",
		Message: fmt.Sprintf(
			"(Used: %s/%s | Reset: %s)",
			fmtutil.PrettyNum(limits.Used),
			fmtutil.PrettyNum(limits.Total),
			timeutil.Format(limits.Reset, "%Y/%m/%d %H:%M:%S"),
		),
	}

	switch {
	case limits.Used == -1:
		chk.Status = support.CHECK_WARN
		chk.Message = "No info about ratelimit"
	case limits.Used >= limits.Total:
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
		fmt.Print(bash.Generate(info, APP))
	case "fish":
		fmt.Print(fish.Generate(info, APP))
	case "zsh":
		fmt.Print(zsh.Generate(info, optMap, APP))
	default:
		return 1
	}

	return 0
}

// printMan prints man page
func printMan() {
	fmt.Println(man.Generate(genUsage(), genAbout("")))
}

// genUsage generates usage info
func genUsage() *usage.Info {
	info := usage.NewInfo()

	if fmtc.Is256ColorsSupported() {
		info.AppNameColorTag = colorTagApp
	}

	info.AddCommand(CMD_DOWNLOAD, "Download and store artefacts", "dir", "?artefact")
	info.AddCommand(CMD_LIST, "List artefacts", "dir/storage")
	info.AddCommand(CMD_GET, "Download artefact", "storage", "name", "?version")
	info.AddCommand(CMD_CLEANUP, "Remove outdated artefacts", "dir", "min-versions")

	info.AddOption(OPT_SOURCES, "Path to YAML file with sources {s-}(default: artefacts.yml){!}", "file")
	info.AddOption(OPT_TOKEN, "GitHub personal token", "token")
	info.AddOption(OPT_INSTALL, "Install artefact to user binaries {s-}($HOME/.bin){!}")
	info.AddOption(OPT_UNIT, "Run application in unit mode {s-}(no colors and animations){!}")
	info.AddOption(OPT_NO_COLOR, "Disable colors in output")
	info.AddOption(OPT_HELP, "Show this help message")
	info.AddOption(OPT_VER, "Show version")

	info.AddExample(
		"download data",
		`Download artefacts to "data" directory`,
	)

	info.AddExample(
		"download data --source ~/artefacts-all.yml",
		`Download artefacts from given file to "data" directory`,
	)

	info.AddExample(
		"download data --name shellcheck",
		`Download shellcheck artefacts to data directory`,
	)

	info.AddExample(
		"list data",
		`List all artefacts in "data" directory`,
	)

	info.AddExample(
		"list my.artefacts.storage",
		`List all artefacts on remote storage`,
	)

	info.AddExample(
		"cleanup data 10",
		`Cleanup artefacts versions in "data" directory except the last 10`,
	)

	info.AddExample(
		"get my.artefacts.storage myapp",
		`Download the latest version of myapp files from remote storage`,
	)

	info.AddExample(
		"get my.artefacts.storage myapp 1.0.0",
		`Download myapp version 1.0.0 files from remote storage`,
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
