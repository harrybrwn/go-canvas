package canvas

import (
	"fmt"
	"net/url"
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
	ch, _ := c.FilesChan()
	// if err := <-errs; err != nil {
	// 	t.Error(err)
	// }
	count := 0
	for f := range ch {
		count++
		fmt.Printf("%d %s\n", f.ID, f.Filename)
	}
	fmt.Println("found", count, "files")
}

func TestCourse(t *testing.T) {
	c := testCourse()
	path := fmt.Sprintf("courses/%d/files", c.ID)
	resp, err := c.client.get(path, url.Values{
		"sort": {"created_at"},
		"page": {"1"},
	})
	if err != nil {
		t.Error(err)
	}
	defer resp.Body.Close()

	// links, err := newLinkedResource(resp)
	// if err != nil {
	// 	t.Error(err)
	// }
	// last := links.links["last"]
	// fmt.Println(last.page)
}

func TestEndpoints(t *testing.T) {

}
