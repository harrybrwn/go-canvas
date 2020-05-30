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
Get a token from your canvas account, [this](https://community.canvaslms.com/docs/DOC-16005-42121018197) should help. Then either set a `$CANVAS_TOKEN` environment variable or using the `canvas.SetToken` or `canvas.New` functions.

### Warning
Everything that is related to the `canvas.Account` struct is untested because I can't figure out how to get access to one for testing.
