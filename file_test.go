package canvas

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"

	"github.com/harrybrwn/errs"
	"github.com/matryer/is"
)

func TestCourse_Files(t *testing.T) {
	is := is.New(t)
	c := testCourse()

	c.SetErrorHandler(func(e error) error {
		t.Fatal(e)
		return e
	})
	is.True(c.client != nil)

	var (
		file   *File
		folder *Folder
	)
	t.Run("Course.Files", func(t *testing.T) {
		is := is.New(t)
		files := c.Files()
		is.True(files != nil)
		for file = range files {
			is.True(file.client != nil)
			is.True(file.ID != 0)
		}
	})

	u, err := file.PublicURL()
	if err != nil {
		t.Error(err)
	}
	if u == "" {
		t.Error("should have gotten a url")
	}

	t.Run("Course.Folders", func(t *testing.T) {
		is := is.New(t)
		folders := c.Folders()
		is.True(folders != nil)
		for folder = range folders {
			is.True(folder.client != nil)
			is.True(folder.ID != 0)
		}
		for f := range folder.Folders() {
			is.True(f.ParentFolderID == folder.ID)
		}
		for f := range folder.Files() {
			is.True(f.FolderID == folder.ID)
		}
	})
}

func TestFiles_Err(t *testing.T) {
	c := testCourse()
	if c.errorHandler == nil {
		t.Error("course should have an error handler")
	}
	c.SetErrorHandler(func(e error) error {
		if e == nil {
			t.Error("expected an error")
		}
		return e
	})

	defer deauthorize(c.client)()
	files := c.Files()
	if files == nil {
		t.Error("nil channel")
	}
	for range files {
		t.Error("should not execure")
	}
}

func TestFolders_Err(t *testing.T) {
	c := testCourse()
	if c.errorHandler == nil {
		t.Error("course should have an error handler")
	}
	c.SetErrorHandler(func(e error) error {
		if e == nil {
			t.Error("expected an error")
		}
		return e
	})
	defer deauthorize(c.client)()
	folders := c.Folders()
	if folders == nil {
		t.Error("nil channel")
	}
	for range folders {
		t.Error("should not execute")
	}
}

func TestCreateFolder(t *testing.T) {
	c := testCourse()
	f, err := c.CreateFolder("/test_folder", IncludeOpt("user"))
	if err != nil {
		t.Error(err)
	}
	if err = f.Delete(); err != nil {
		t.Error(err)
	}
}

func TestFilesFolders(t *testing.T) {
	c := testCourse()
	folder, err := c.Folder(19926068)
	if err != nil {
		t.Error(err)
	}
	byPath, err := FolderPath("/testfolder/another")
	if len(byPath) != 3 {
		t.Error("expected three folders")
	}

	parent, err := folder.ParentFolder()
	if err != nil {
		t.Error(err)
	}
	_, err = parent.ParentFolder()
	if err == nil {
		t.Error("the root folder has no parent")
	}
	f, err := folder.ParentFolder()
	if f != parent {
		t.Error("should be the same pointer")
	}

	file, err := parent.File(95954272)
	if err != nil {
		t.Error(err)
	}
	folder, err = file.ParentFolder()
	if err != nil {
		t.Error(err)
	}
	f, _ = file.ParentFolder()
	if f != folder {
		t.Error("pointers should be the same")
	}
	files := Files(ContentTypes("application/x-yaml", "text/markdown"))
	for file = range files {
		if file.ContentType != "application/x-yaml" && file.ContentType != "text/markdown" {
			t.Error("got wrong content type")
		}
	}
}

func TestFileUpload(t *testing.T) {
	osfile, err := os.Open("./README.md")
	if err != nil {
		t.Fatal(err)
	}
	defer osfile.Close()
	stats, err := osfile.Stat()
	if err != nil {
		t.Error(err)
	}

	file, err := UploadFile(
		"readme.md", osfile,
		ContentType("text/markdown"),
		Opt("size", stats.Size()),
		Opt("on_duplicate", "overwrite"),
		Opt("no_redirect", true),
		Opt("parent_folder_path", "/"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if file == nil {
		t.Fatal("got nil response file")
	}
	baseid := file.FolderID
	newname := "The_ReadMe_file.md"
	err = file.Rename(newname)
	if err != nil {
		t.Error(err)
	}
	if file.Name() != newname {
		t.Errorf("name was not updated from %s to %s", file.Name(), newname)
	}
	if err = file.Move(&Folder{FullName: "/testfolder"}); err != nil {
		t.Error(err)
	}
	if err = file.Move(&Folder{ID: baseid}); err != nil {
		t.Error(err)
	}

	if err = file.Delete(); err != nil {
		t.Error(err)
	}
	_, err = GetFile(file.ID)
	if err == nil {
		t.Error("expected an error here")
	}
}

func newSyncAuth(a *auth, mu *sync.Mutex) *syncAuth {
	return &syncAuth{
		tok: a.token,
		rt:  http.DefaultTransport,
		mu:  mu,
	}
}

type syncAuth struct {
	tok string
	rt  http.RoundTripper
	mu  *sync.Mutex
}

func (sa *syncAuth) RoundTrip(r *http.Request) (*http.Response, error) {
	sa.mu.Lock()
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sa.tok))
	r.Header.Set("User-Agent", "tests: "+DefaultUserAgent)
	r.Host = DefaultHost
	r.URL.Host = DefaultHost
	r.URL.Scheme = "https"
	resp, err := sa.rt.RoundTrip(r)
	sa.mu.Unlock()
	return resp, err
}

func TestPaginationErrors(t *testing.T) {
	c := testCourse()
	tr := c.client.(*http.Client).Transport
	var mu sync.Mutex
	// we need a syncAuth for this test because some goroutines will be
	// modifying the client object
	if a, ok := tr.(*auth); ok {
		c.client.(*http.Client).Transport = newSyncAuth(a, &mu)
	}
	allfiles, err := c.ListFiles()
	if err != nil {
		t.Fatal(err)
	}
	var testerror = errs.New("test error")

	t.Run("send_error", func(t *testing.T) {
		readCount := 0
		ch := make(fileChan)
		send := func(r io.Reader) error {
			mu.Lock()
			readCount++
			if readCount == 4 {
				mu.Unlock()
				return testerror // send an error only after the first request
			}
			mu.Unlock()
			files := make([]*File, 0)
			err := json.NewDecoder(r).Decode(&files)
			for _, f := range files {
				ch <- f
			}
			return err
		}
		p := newPaginatedList(
			c.client, fmt.Sprintf("courses/%d/files/", c.ID),
			send, nil,
		)
		p.perpage = 4
		go handleErrs(p.start(), ch, func(e error) error {
			if e != testerror {
				t.Error("should only be handling the error I sent")
			}
			return nil
		})
		fileCount := 0
		for range ch {
			fileCount++
		}
		if readCount != 5 {
			t.Error("should have gone through all the pages")
		}
		if fileCount >= len(allfiles) {
			t.Error("should not have gotten all of the files")
		}
	})
	t.Run("auth_error", func(t *testing.T) {
		var tok string
		readCount := 0
		ch := make(fileChan)
		send := func(r io.Reader) error {
			mu.Lock()
			readCount++
			if readCount == 2 {
				tok = c.client.(*http.Client).Transport.(*syncAuth).tok
				c.client.(*http.Client).Transport.(*syncAuth).tok = ""
			}
			mu.Unlock()
			files := make([]*File, 0)
			err := json.NewDecoder(r).Decode(&files)
			for _, f := range files {
				ch <- f
			}
			return err
		}
		p := newPaginatedList(
			c.client,
			fmt.Sprintf("courses/%d/files/", c.ID),
			send, nil,
		)
		p.perpage = 4
		go handleErrs(p.start(), ch, func(e error) error {
			if e == nil {
				t.Error("expected error")
			}
			err, ok := e.(*AuthError)
			if !ok {
				t.Errorf("expected an auth error; got %T", err)
			}
			return nil
		})
		count := 0
		for f := range ch {
			if f.ID == 0 {
				t.Error("got bad file id")
			}
			count++
		}
		c.client.(*http.Client).Transport.(*syncAuth).tok = tok
		if count >= len(allfiles) {
			t.Error("should not have the same count")
		}
	})
}
