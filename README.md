<p align="center"><a href="#readme"><img src="https://gh.kaos.st/artefactor.png" /></a></p>

<p align="center">
  <a href="https://kaos.sh/w/artefactor/ci"><img src="https://kaos.sh/w/artefactor/ci.svg" alt="GitHub Actions CI Status" /></a>
  <a href="https://kaos.sh/l/artefactor"><img src="https://kaos.sh/l/8ce6d02d3ef53e599745.svg" alt="Code Climate Maintainability" /></a>
  <a href="https://kaos.sh/b/artefactor"><img src="https://kaos.sh/b/10e3bcb5-b2bb-4837-9482-6824e5934f98.svg" alt="Codebeat badge" /></a>
  <a href="https://kaos.sh/w/artefactor/codeql"><img src="https://kaos.sh/w/artefactor/codeql.svg" alt="GitHub Actions CodeQL Status" /></a>
  <a href="#license"><img src="https://gh.kaos.st/apache2.svg"></a>
</p>

<p align="center"><a href="#usage-demo">Usage demo</a> • <a href="#installation">Installation</a> • <a href="#usage">Usage</a> • <a href="#ci-status">CI Status</a> • <a href="#contributing">Contributing</a> • <a href="#license">License</a></p>

<br/>

`artefactor` is utility for downloading artefacts from GitHub.

### Usage demo

[![demo](https://gh.kaos.st/artefactor-020.gif)](#usage-demo)

### Installation

#### From source

To build the `artefactor` from scratch, make sure you have a working Go 1.21+ workspace (_[instructions](https://go.dev/doc/install)_), then:

```bash
go install github.com/essentialkaos/artefactor@latest
```

#### Prebuilt binaries

You can download prebuilt binaries for Linux from [EK Apps Repository](https://apps.kaos.st/artefactor/latest):

```bash
bash <(curl -fsSL https://apps.kaos.st/get) artefactor
```

### Usage

```
Usage: artefactor {options} {command}

Commands

  download dir artefact       Download and store artefacts
  list dir/storage            List artefacts
  get storage name version    Download artefact
  cleanup dir min-versions    Remove outdated artefacts

Options

  --sources, -s file    Path to YAML file with sources (default: artefacts.yml)
  --token, -t token     GitHub personal token
  --install, -I         Install artefact to user binaries ($HOME/.bin)
  --unit, -u            Run application in unit mode (no colors and animations)
  --no-color, -nc       Disable colors in output
  --help, -h            Show this help message
  --version, -v         Show version

Examples

  artefactor download data
  Download artefacts to "data" directory

  artefactor download data --source ~/artefacts-all.yml
  Download artefacts from given file to "data" directory

  artefactor download data --name shellcheck
  Download shellcheck artefacts to data directory

  artefactor list data
  List all artefacts in "data" directory

  artefactor list my.artefacts.storage
  List all artefacts on remote storage

  artefactor cleanup data 10
  Cleanup artefacts versions in "data" directory except the last 10

  artefactor get my.artefacts.storage myapp
  Download the latest version of myapp files from remote storage

  artefactor get my.artefacts.storage myapp 1.0.0
  Download myapp version 1.0.0 files from remote storage
```

### CI Status

| Branch | Status |
|--------|----------|
| `master` | [![CI](https://kaos.sh/w/artefactor/ci.svg?branch=master)](https://kaos.sh/w/artefactor/ci?query=branch:master) |
| `develop` | [![CI](https://kaos.sh/w/artefactor/ci.svg?branch=develop)](https://kaos.sh/w/artefactor/ci?query=branch:develop) |

### Contributing

Before contributing to this project please read our [Contributing Guidelines](https://github.com/essentialkaos/contributing-guidelines#contributing-guidelines).

### License

[Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0)

<p align="center"><a href="https://essentialkaos.com"><img src="https://gh.kaos.st/ekgh.svg"/></a></p>
