package canvas_test

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/harrybrwn/go-canvas"
)

func Example_concurrentErrorHandling() {
	var (
		failed bool
		err    error
	)
	canvas.SetToken("bad token")
	reset := canvas.ConcurrentErrorHandler
	canvas.ConcurrentErrorHandler = func(e error) error {
		if _, ok := e.(*canvas.Error); !ok {
			failed = true
			err = e
			return e // non-nil will stop all goroutines
		}
		return nil // nil means we want to continue
	}

	count := 0
	for file := range canvas.Files() {
		if failed {
			// log.Fatal(err)
			break
		}
		if file != nil {
			count++
		}
	}
	canvas.ConcurrentErrorHandler = reset
	fmt.Println(err)
	fmt.Println(failed)

	// Output:
	// Invalid access token.
	// true

	canvas.SetToken(os.Getenv("CANVAS_TEST_TOKEN"))
}

func ExampleFile_AsWriteCloser() {
	file := canvas.NewFile("test-file")
	fmt.Println(file.ID == 0)

	wc, err := file.AsWriteCloser()
	if err != nil {
		log.Fatal("could not create io.WriteCloser:", err)
	}
	if _, err = io.WriteString(wc, "this is a test file for the examples"); err != nil {
		log.Fatal("could not write data:", err)
	}
	// close sends the data to canvas and updates the 'file' pointer
	if err = wc.Close(); err != nil {
		log.Fatal("could not send data: ", err)
	}
	fmt.Println(file.ID == 0)

	// Output:
	// true
	// false

	file.Delete()
}
