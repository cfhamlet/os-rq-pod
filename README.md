# os-rq-pod

[![Build Status](https://www.travis-ci.org/cfhamlet/os-rq-pod.svg?branch=master)](https://www.travis-ci.org/cfhamlet/os-rq-pod)
[![codecov](https://codecov.io/gh/cfhamlet/os-rq-pod/branch/master/graph/badge.svg)](https://codecov.io/gh/cfhamlet/os-rq-pod)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/cfhamlet/os-rq-pod?tab=overview)

Request queue pod for broad crawls


## Install

You can get the library with ``go get``

```
go get -u github.com/cfhamlet/os-rq-pod
```

The binary command line tool can be build from source, you should always build the latest released version , [Releases List](https://github.com/cfhamlet/os-rq-pod/releases)

```
git clone -b v0.0.1 https://github.com/cfhamlet/os-rq-pod.git
cd os-rq-pod
make install
```

## Usage

### Command line

```
$ rq-pod -h
rq-pod command line tool

Usage:
  rq-pod [flags]
  rq-pod [command]

Available Commands:
  help        Help about any command
  run         run rq-pod server
  version     show version info

Flags:
  -h, --help   help for rq-pod

Use "rq-pod [command] --help" for more information about a command.
```

## License
  MIT licensed.

