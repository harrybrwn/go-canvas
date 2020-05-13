package canvas

import (
	"errors"
	"os"
	"testing"
)

func testToken() string {
	tok := os.Getenv("CANVAS_TOKEN")
	if tok == "" {
		panic("no testing token")
	}
	return tok
}

var testingUser *User

func testUser(c *Canvas) *User {
	var err error
	if testingUser == nil {
		if testingUser, err = c.CurrentUser(); err != nil {
			panic("Error getting test User: " + err.Error())
		}
	}
	return testingUser
}

func TestCanvas(t *testing.T) {
	c := New(testToken())
	if c == nil {
		t.Fatal("get nil canvas object")
	}
	u := testUser(c)
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
