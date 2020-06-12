[![Build Status](https://api.travis-ci.org/kszab0/serve.svg?branch=master)](https://travis-ci.org/github/kszab0/serve)
[![Coverage Status](https://coveralls.io/repos/github/kszab0/serve/badge.svg?branch=master)](https://coveralls.io/github/kszab0/serve?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/kszab0/serve)](https://goreportcard.com/report/github.com/kszab0/serve)

# serve

**serve** is a simple cross-platform CLI tool to serve files over HTTP.

## Install
You must have [Go](https://golang.org/) installed in order to build **serve**.

```
go get github.com/kszab0/serve
```

## Usage
```
serve path/to/serve
```
Defaults to the current directory.

```
Usage of serve:
  -a string
        http address (default "localhost:9876")
  -q    use quiet mode - don't display logs
```

## License
Authored by [Kristóf Szabó](mailto:kristofszabo@protonmail.com) and released under the MIT license.