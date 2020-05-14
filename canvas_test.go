package canvas

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/matryer/is"
)

func testToken() string {
	tok := os.Getenv("CANVAS_TOKEN")
	if tok == "" {
		panic("no testing token")
	}
	return tok
}

var (
	testingUser    *User
	testingCourses []*Course
)

func testUser() (*User, error) {
	var err error
	if testingUser == nil {
		c := New(testToken())
		testingUser, err = c.CurrentUser()
	}
	return testingUser, err
}

func testCourses() ([]*Course, error) {
	var err error
	if testingCourses == nil {
		c := New(testToken())
		testingCourses, err = c.Courses()
	}
	return testingCourses, err
}

func Test(t *testing.T) {
	is := is.New(t)

	c := New(testToken())
	// u, err := c.CurrentUser()
	// is.NoErr(err)
	cal, err := c.CalendarEvents()
	is.NoErr(err)
	fmt.Println(cal)
}

func TestCanvas(t *testing.T) {
	t.Skip("don't really nead this test")
	c := New(testToken())
	if c == nil {
		t.Fatal("get nil canvas object")
	}
	u, err := testUser()
	if err != nil {
		panic("could not get test user: " + err.Error())
	}
	if u == nil {
		t.Error("got nil user")
	}
	if u.client == nil {
		t.Error("user has no client")
	}
	courses, err := c.CompletedCourses()
	if err != nil {
		t.Error(err)
	}
	for _, crs := range courses {
		if crs.client == nil {
			t.Error("course should have gotten a client")
		}
		if crs.errorHandler == nil {
			t.Error("course should have gotten an error handling function")
		}
	}
}

func TestCanvas_Err(t *testing.T) {
	for _, c := range []*Canvas{
		WithHost(testToken(), ""),
		WithHost("", DefaultHost),
	} {
		_, err := c.CurrentUser()
		if err == nil {
			t.Error("expected an error")
		}
		courses, err := c.ActiveCourses()
		if err == nil {
			t.Error("expected an error")
		}
		if courses != nil {
			t.Error("expected nil courses")
		}
	}
}

func TestUser(t *testing.T) {
	t.Skip()
	is := is.New(t)
	u, err := testUser()
	is.NoErr(err)
	settings, err := u.Settings()
	is.NoErr(err)
	is.True(len(settings) > 0)

	profile, err := u.Profile()
	is.NoErr(err)
	is.True(profile.ID != 0)
	is.True(len(profile.Name) > 0)

	subs, err := u.GradedSubmissions()
	is.NoErr(err)
	is.True(len(subs) > 0)

	colors, err := u.Colors()
	is.NoErr(err)
	var col, val string
	for col, val = range colors {
		break
	}
	color, err := u.Color(col)
	is.NoErr(err)
	is.Equal(color.HexCode, val)
}

func TestUser_Err(t *testing.T) {
	is := is.New(t)
	u, err := testUser()
	is.NoErr(err)
	colors, err := u.Colors()
	is.NoErr(err)
	defer deauthorize(u.client)()

	settings, err := u.Settings()
	is.True(err != nil)
	is.True(settings == nil)
	is.True(len(settings) == 0)

	profile, err := u.Profile()
	is.True(err != nil)
	is.True(profile == nil)

	var col string
	for col = range colors {
		break
	}
	color, err := u.Color(col)
	is.True(err != nil)
	is.True(color == nil)

	err = u.SetColor(col, "#FFFFFF")
	is.True(err != nil)
	_, ok := err.(*AuthError)
	is.True(ok)
}

func TestCourse_Files(t *testing.T) {
	is := is.New(t)
	courses, err := testCourses()
	is.NoErr(err)
	c := courses[0]
	c.SetErrorHandler(func(e error, quit chan int) {
		t.Fatal(e)
		quit <- 1
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

	t.Run("Course.Folders", func(t *testing.T) {
		is := is.New(t)
		folders := c.Folders()
		is.True(folders != nil)
		for folder = range folders {
			is.True(folder.client != nil)
			is.True(folder.ID != 0)
		}
	})

	for file = range folder.Files() {
		is.True(file.FolderID == folder.ID)
	}
	parent, err := file.Folder()
	is.NoErr(err)
	is.True(parent.ID == folder.ID)

	t.Run("Course.Folder", func(t *testing.T) {
		is := is.New(t)
		f, err := c.Folder(parent.ID)
		is.NoErr(err)
		is.True(f.ID == parent.ID)
		is.True(f.ID == file.FolderID)
	})
	t.Run("Course.File", func(t *testing.T) {
		is := is.New(t)
		f, err := c.File(file.ID)
		is.NoErr(err)
		is.True(f.ID == file.ID)
		is.True(f.DisplayName == file.DisplayName)
	})
}

func TestCourseFiles_Err(t *testing.T) {
	is := is.New(t)
	courses, err := testCourses()
	is.NoErr(err)
	c := courses[1]
	errorCount := 0
	c.SetErrorHandler(func(e error, q chan int) {
		if e == nil {
			t.Error("expected an error")
		} else {
			errorCount++
		}
		q <- 1
	})

	t.Run("Files", func(t *testing.T) {
		is := is.New(t)
		all, err := c.ListFiles()
		is.NoErr(err)
		i := 0
		files := c.Files()
		defer deauthorize(c.client)() // deauthorize after goroutines started
		for f := range files {
			is.True(f.ID != 0) // these be valid
			i++
		}
		is.True(len(all) > i) // the channel should have been stopped early
		files = c.Files()
		is.True(files != nil)
		for range files {
			panic("this code should not execute")
		}
	})

	t.Run("Folders", func(t *testing.T) {
		is := is.New(t)
		all, err := c.ListFolders()
		is.NoErr(err)
		i := 0
		folders := c.Folders()
		defer deauthorize(c.client)()
		for f := range folders {
			is.True(f.ID > 0)
			is.True(f.ID == all[i].ID)
			i++
		}
		is.True(len(all) >= i)
		for range folders {
			panic("this code should not execute")
		}
	})
	is.Equal(errorCount, 2)
}

func TestErrChan(t *testing.T) {
	is := is.New(t)
	courses, err := testCourses()
	is.NoErr(err)
	c := courses[1]
	files, _ := c.FilesErrChan()
	for range files {
	}
	folders, _ := c.FoldersErrChan()
	for range folders {
	}
}

func TestErrPair(t *testing.T) {
	tt := []struct {
		err error
		exp string
	}{
		{errpair(errors.New("one"), errors.New("two")), "one, two"},
		{errpair(errors.New("one"), nil), "one"},
		{errpair(nil, errors.New("two")), "two"},
	}
	for i, tc := range tt {
		if tc.err.Error() != tc.exp {
			t.Errorf("test case %d for errpair gave wrong result", i)
		}
	}
	err := errpair(nil, nil)
	if err != nil {
		t.Error("a pair of nil errors should result in one nil error")
	}
}

func TestErrors(t *testing.T) {
	is := is.New(t)
	e := &AuthError{
		Status: "test",
		Errors: []errorMsg{{"one"}, {"two"}},
	}
	is.Equal(e.Error(), "test: one, two")
	e = &AuthError{
		Status: "",
		Errors: []errorMsg{{"one"}, {"two"}},
	}
	is.Equal(e.Error(), "one, two")
	is.Equal(checkErrors([]errorMsg{}), "")
}

func deauthorize(d doer) func() {
	cli, ok := d.(*client)
	if !ok {
		return func() {}
	}
	au, ok := cli.Transport.(*auth)
	if !ok {
		return func() {}
	}
	token := au.token
	// remove the token
	au.token = ""
	return func() {
		au.token = token
	}
}
