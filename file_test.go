package canvas

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/matryer/is"
)

var courseRoot *Folder

func testCourseRoot() *Folder {
	var err error
	if courseRoot == nil {
		c := testCourse()
		courseRoot, err = c.Root()
		if err != nil {
			panic(err)
		}
	}
	return courseRoot
}

func TestFolders(t *testing.T) {
	is := is.New(t)
	folder := NewFolder("test")
	if folder.Foldername != "test" {
		t.Error("wrong foldername")
	}
	if folder.client == nil {
		t.Error("needs client")
	}
	ConcurrentErrorHandler = func(e error) error {
		fmt.Println("Error in Testing:", e)
		return e
	}
	cli, mux, server := testServer()
	defer server.Close()
	defer swapCanvas(&Canvas{client: cli})()
	mux.HandleFunc(fmt.Sprintf("%s/users/self/folders", apiPath), handlePagingatedList(t, 3, "folder.json"))
	nfiles := 5
	mux.HandleFunc(fmt.Sprintf("%s/users/self/files", apiPath), handlePagingatedList(t, nfiles, "file.json"))

	i := 0
	for f := range Folders() {
		i++
		if f.ID != 2937 {
			t.Error("did not get 2937 as folder id")
		}
	}
	is.Equal(i, 3) // should have 3 folders
	i = 0
	for f := range Files() {
		i++
		is.Equal(f.ID, 569) // should have testing id
	}
	is.Equal(i, nfiles)
	files, err := ListFiles()
	is.NoErr(err)
	is.Equal(len(files), nfiles)
	folders, err := ListFolders()
	is.NoErr(err)
	is.Equal(len(folders), 3)
}

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

			dir, name := filepath.Split(folder.FullName)
			is.Equal(dir, folder.Path())
			is.Equal(name, folder.Name())
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
		t.Error("should not execute")
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
	client, mux, server := testServer()
	defer server.Close()
	mux.HandleFunc("/api/v1/users/self/folders", func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "POST")
		q := r.URL.Query()
		if q.Get("include[]") != "user" {
			t.Error("expected user param")
		}
		name := q.Get("name")
		parent := q.Get("parent_folder_path")
		if parent != "/" {
			t.Error("should have root folder in params list")
		}
		if name != "testfolder" {
			t.Error("wrong folder name")
		}
		w.Write([]byte(fmt.Sprintf(`{"id":11,"name":"%s","full_name":"%s"}`, name, path.Join(parent, name))))
	})
	defer swapCanvas(&Canvas{client: client})()
	f, err := CreateFolder("/testfolder", IncludeOpt("user"))
	if err != nil {
		t.Error(err)
	}
	if f.ID != 11 {
		t.Error("wrong id")
	}
	if f.Foldername != "testfolder" {
		t.Error("responded with wrong folder name")
	}
}

func TestFolderPath(t *testing.T) {
	fs, err := FolderPath("/")
	if err != nil {
		t.Error(err)
	}
	if len(fs) < 1 {
		t.Fatalf("folder path length should be 1 not %d", len(fs))
	}
	folder := fs[0]
	for f := range folder.Files() {
		if f.folder != folder {
			t.Error("did not save folder")
		}
		if f.Path() != folder.FullName {
			t.Error("got wrong path")
		}
	}
}

func TestRoot(t *testing.T) {
	f := testCourseRoot()
	if f.Name() != "course files" {
		t.Error("this is the wrong folder")
	}

	u, err := testUser()
	if err != nil {
		t.Error(err)
	}
	f, err = u.Root()
	if err != nil {
		t.Error(err)
	}
	if f.Name() != "my files" {
		t.Error("got the wrong folder")
	}
	root, err := Root()
	if err != nil {
		t.Error(err)
	}
	if f.ID != root.ID {
		t.Error("these should be the same folder")
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
		t.Errorf("expected 3 folders; got %d", len(byPath))
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
}

func TestFile_AsWriteCloser(t *testing.T) {
	file := NewFile("test-file")

	wc, err := file.AsWriteCloser()
	if err != nil {
		t.Error("could not create io.WriteCloser:", err)
	}
	if _, err = io.WriteString(wc, "this is a test file for the examples"); err != nil {
		t.Error("could not write data:", err)
	}
	// close sends the data to canvas and updates the 'file' pointer
	if wc.(*fileWriter).d == nil {
		t.Error("write closer should have a doer")
	}
	if err = wc.Close(); err != nil {
		t.Fatal("could not send data:", err)
	}
	defer file.Delete()

	newfile, err := GetFile(file.ID)
	if err != nil {
		t.Error(err)
	}
	if newfile.ID != file.ID {
		t.Error("got wrong file ids:", newfile.ID, file.ID)
	}
	rc, err := newfile.AsReadCloser()
	if err != nil {
		t.Error("could not create an io.ReadCloser from the file:", err)
	}
	b := new(bytes.Buffer)
	if _, err = b.ReadFrom(rc); err != nil {
		t.Error("could not read from file:", err)
	}
	if b.String() != "this is a test file for the examples" {
		t.Error("did not get the correct file contents")
	}
	b.Reset()
	if _, err = newfile.WriteTo(b); err != nil {
		t.Error(err)
	}
	if b.String() != "this is a test file for the examples" {
		t.Error("did not get the correct file contents")
	}
}

func TestFolder_Copy(t *testing.T) {
	c := testCourse()
	paths, err := c.FolderPath("/apizza/pkg/cache")
	if err != nil {
		t.Error("FolderPath failed:", err)
	}
	l := len(paths)
	folder := paths[l-1]
	dest := paths[1]
	if err = folder.Copy(dest); err != nil {
		t.Error(err)
	}
	paths, err = c.FolderPath("/apizza/cache")
	if err != nil {
		t.Error(err)
	}
	if len(paths) < 3 {
		t.Fatal("did not copy folder")
	}
	if err = paths[len(paths)-1].Delete(); err != nil {
		t.Error(err)
	}
}

func foldersHandlerFunc(t *testing.T, n int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", `<https://canvas.instructure.com/api/v1/courses/000/users?search_term=test&page=1&per_page=10>; rel="current",<https://canvas.instructure.com/api/v1/courses/000/users?search_term=test&page=1&per_page=10>; rel="first",<https://canvas.instructure.com/api/v1/courses/000/users?search_term=test&page=1&per_page=10>; rel="last"`)
		w.WriteHeader(200)
		w.Write([]byte("["))
		for i := 0; i < n; i++ {
			writeTestFile(t, "folder.json", w)
			if i < n-1 {
				w.Write([]byte(","))
			}
		}
		w.Write([]byte("]"))
	}
}

func filesHandlerFunc(t *testing.T, n int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", `<https://canvas.instructure.com/api/v1/courses/000/users?search_term=test&page=1&per_page=10>; rel="current",<https://canvas.instructure.com/api/v1/courses/000/users?search_term=test&page=1&per_page=10>; rel="first",<https://canvas.instructure.com/api/v1/courses/000/users?search_term=test&page=1&per_page=10>; rel="last"`)
		w.WriteHeader(200)
		w.Write([]byte("["))
		for i := 0; i < n; i++ {
			writeTestFile(t, "file.json", w)
			if i < n-1 {
				w.Write([]byte(","))
			}
		}
		w.Write([]byte("]"))
	}
}

func handlePagingatedList(t *testing.T, n int, file string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", `<https://canvas.instructure.com/api/v1/path/?&page=1&per_page=10>; rel="current",<https://canvas.instructure.com/api/v1/path?page=1&per_page=10>; rel="first",<https://canvas.instructure.com/api/v1/path?page=1&per_page=10>; rel="last"`)
		w.WriteHeader(200)
		w.Write([]byte("["))
		for i := 0; i < n; i++ {
			writeTestFile(t, file, w)
			if i < n-1 {
				w.Write([]byte(","))
			}
		}
		w.Write([]byte("]"))
	}
}
