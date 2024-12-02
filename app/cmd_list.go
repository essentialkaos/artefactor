package app

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"strings"

	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/fmtutil"
	"github.com/essentialkaos/ek/v13/fsutil"
	"github.com/essentialkaos/ek/v13/jsonutil"
	"github.com/essentialkaos/ek/v13/options"
	"github.com/essentialkaos/ek/v13/pager"
	"github.com/essentialkaos/ek/v13/path"
	"github.com/essentialkaos/ek/v13/req"
	"github.com/essentialkaos/ek/v13/terminal"
	"github.com/essentialkaos/ek/v13/terminal/tty"

	"github.com/essentialkaos/artefactor/data"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// cmdList is "list" command handler
func cmdList(args options.Arguments) error {
	if !args.Has(0) {
		return fmt.Errorf("You must provide path to data directory or URL of storage")
	}

	var err error
	var index *data.Index

	switch {
	case fsutil.IsExist(args.Get(0).Clean().String()):
		index, err = readLocalIndex(args.Get(0).Clean().String())
	case strings.Contains(args.Get(0).String(), "."):
		index, err = readRemoteIndex(args.Get(0).String())
	default:
		return fmt.Errorf("Invalid data directory or URL")
	}

	if err != nil {
		return fmt.Errorf("Can't get index data: %v", err)
	} else if index.IsEmpty() {
		terminal.Warn("No artefacts found")
		return nil
	}

	if tty.IsTTY() {
		if pager.Setup() == nil {
			defer pager.Complete()
		}
	}

	for _, info := range index.Artefacts {
		size := fmtutil.PrettyNum(len(info.Versions))

		fmtc.Printfn("{s-}┌{!}{*@} %s {!}{#240}{*@} %s {!}", info.Name, size)
		fmtc.Printfn("{s-}│{!}")

		for i, version := range info.Versions {
			if i+1 != len(info.Versions) {
				fmtc.Printfn(
					"{s-}├{!} {s}%s{!} {s-}(%s){!}",
					version.Version,
					fmtutil.PrettySize(version.Size),
				)
			} else {
				fmtc.Printfn(
					"{s-}└{!} {*}%s{!} {s-}(%s){!}",
					version.Version,
					fmtutil.PrettySize(version.Size),
				)
			}
		}

		fmtc.NewLine()
	}

	return nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// readLocalIndex reads index from filesystem
func readLocalIndex(dir string) (*data.Index, error) {
	indexFile := path.Join(dir, "index.json")
	err := fsutil.ValidatePerms("FRS", indexFile)

	if err != nil {
		return nil, err
	}

	index := &data.Index{}
	err = jsonutil.Read(indexFile, index)

	if err != nil {
		return nil, err
	}

	return index, nil
}

// readRemoteIndex reads index from remote storage
func readRemoteIndex(url string) (*data.Index, error) {
	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}

	resp, err := req.Request{
		URL:         url + "/index.json",
		Accept:      req.CONTENT_TYPE_JSON,
		ContentType: req.CONTENT_TYPE_JSON,
		AutoDiscard: true,
	}.Get()

	if err != nil {
		return nil, fmt.Errorf("Can't send request: %v", err)
	} else if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Storage returned non-ok status code %d", resp.StatusCode)
	}

	index := &data.Index{}
	err = resp.JSON(index)

	if err != nil {
		return nil, fmt.Errorf("Can't decode index: %v", err)
	}

	return index, nil
}
