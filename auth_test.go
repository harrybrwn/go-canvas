package canvas

import (
	"fmt"
	"os"
	"sync"
	"testing"
)

func testToken() string {
	return os.Getenv("CANVAS_TOKEN")
}

var (
	createCanvasOnce = sync.Once{}
	testingCanvas    *Canvas
	testingCourse    *Course
)

func testCanvas() *Canvas {
	createCanvasOnce.Do(func() {
		testingCanvas = FromToken(testToken())
	})
	if testingCanvas == nil {
		panic("could not create or find canvas object for testing")
	}
	return testingCanvas
}

func testCourse() *Course {
	if testingCourse == nil {
		cs, err := testCanvas().ActiveCourses()
		if err != nil {
			panic(err)
		}
		testingCourse = cs[1]
	}
	return testingCourse
}

func TestAuth(t *testing.T) {
	c := testCourse()
	files := c.Files(
		ContentType("application/pdf"),
		SortOpt("created_at", "size"),
	)
	for f := range files {
		fmt.Println(f.CreatedAt, f.Size, f.Filename)
	}
	// p := makeparams(
	// 	ArrayOpt("content_type")
	// )
	// resp, err := get(c.client, c.filespath(), p)
	// if err != nil {
	// }
}
