# go-canvas
An API client for Instructure's Canvas API.

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
3. For more advance usage, viewing the [canvas API docs](https://canvas.instructure.com/doc/api/index.html) and using the `canvas.Option` interface will be usful for more fine-tuned api use.

### Concurrent Error Handling
Error handling for functions that return a channel and no error is done with a callback. This callback is called `ConcurrentErrorHandler` and in some cases, a struct may have a `SetErrorHandler` function.
```go
canvas.ConcurrentErrorHandler = func(e error) error {
    if canvas.IsRateLimit(e) {
        fmt.Println("rate limit reached")
        return nil
    }
    return e
}
for f := range canvas.Files() {
    fmt.Println(f.Filename, f.ID)
}
```

## TODO
* Groups
* Outcome Groups
* Favorites
* Submissions
    * submiting assignments
    * file upload on assginments
