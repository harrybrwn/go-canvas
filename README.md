# go-canvas
A (very incomplete) api client for Instructure's Canvas API.

[![Build Status](https://travis-ci.com/harrybrwn/go-canvas.svg?branch=master)](https://travis-ci.com/harrybrwn/go-canvas)
[![GoDoc](https://godoc.org/github.com/github.com/harrybrwn/go-canvas?status.svg)](https://pkg.go.dev/github.com/harrybrwn/go-canvas?tab=doc)
[![Go Report Card](https://goreportcard.com/badge/github.com/harrybrwn/go-canvas)](https://goreportcard.com/report/github.com/harrybrwn/go-canvas)
[![codecov](https://codecov.io/gh/harrybrwn/go-canvas/branch/master/graph/badge.svg)](https://codecov.io/gh/harrybrwn/go-canvas)
[![TODOs](https://badgen.net/https/api.tickgit.com/badgen/github.com/harrybrwn/go-canvas)](https://www.tickgit.com/browse?repo=github.com/harrybrwn/go-canvas)

## Download/Install
```
go get github.com/harrybrwn/go-canvas
```

## Getting Started
1. Get a token from your canvas account, [this](https://community.canvaslms.com/docs/DOC-16005-42121018197) should help.
2. Give the token to the library
    * Set `$CANVAS_TOKEN` environment variable
    * Call `canvas.SetToken` or `canvas.New`

### Warning
Everything that is related to the `canvas.Account` struct is not as extensivly tested because I can't figure out how to get access to one for testing.

# TODO
* Groups
* Outcome Groups
* Favorites
* Submissions
    * assignments
    * file uploads
