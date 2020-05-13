package canvas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// New will create a Canvas struct from an api token.
// New uses the default host.
func New(token string) *Canvas {
	return &Canvas{
		client: newclient(token),
	}
}

// WithHost will create a canvas object that uses a
// different hostname.
func WithHost(token, host string) *Canvas {
	c := &Canvas{client: &client{http.Client{}}}
	authorize(&c.client.Client, token, host)
	return c
}

// Canvas is the main api controller.
type Canvas struct {
	client *client
}

// Courses lists all of the courses associated
// with that canvas object.
func (c *Canvas) Courses(opts ...Option) ([]*Course, error) {
	return c.getCourses(asParams(opts))
}

// ActiveCourses returns a list of only the courses that are
// currently active
func (c *Canvas) ActiveCourses(options ...string) ([]*Course, error) {
	return c.getCourses(&url.Values{
		"enrollment_state": {"active"},
		"include[]":        options,
	})
}

// CompletedCourses returns a list of only the courses that are
// not currently active and have been completed
func (c *Canvas) CompletedCourses(options ...string) ([]*Course, error) {
	return c.getCourses(&url.Values{
		"enrollment_state": {"completed"},
		"include[]":        options,
	})
}

// GetUser will return a user object given that user's ID.
func (c *Canvas) GetUser(id int, opts ...Option) (*User, error) {
	return c.getUser(id, opts)
}

// CurrentUser get the currently logged in user.
func (c *Canvas) CurrentUser(opts ...Option) (*User, error) {
	return c.getUser("self", opts)
}

// pathVar is an interface{} because internally, either "self" or some integer id
// will be passed to be used as an api path parameter.
func (c *Canvas) getUser(pathVar interface{}, opts []Option) (*User, error) {
	res := struct {
		*User
		Errors []*errorMsg
	}{&User{client: c.client}, nil}

	if err := getjson(
		c.client, &res,
		fmt.Sprintf("users/%v", pathVar),
		asParams(opts),
	); err != nil {
		return nil, err
	}
	return res.User, checkErrors(res.Errors)
}

// CurrentUserTodo will get the current user's todo's.
func (c *Canvas) CurrentUserTodo() error {
	panic("not implimented")
}

func (c *Canvas) getCourses(vals encoder) ([]*Course, error) {
	crs := make([]*Course, 0)
	resp, err := get(c.client, "courses", vals)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// read the response into a buffer
	var b bytes.Buffer
	if _, err = b.ReadFrom(resp.Body); err != nil {
		return nil, err
	}

	// try to unmarshal the response into a course array
	// if it fails then we unmarshal it into an error
	if err = json.Unmarshal(b.Bytes(), &crs); err != nil {
		e := &authError{}
		err = json.Unmarshal(b.Bytes(), e)
		return nil, errpair(e, err) // return any and all non-nil errors
	}

	for i := range crs {
		crs[i].client = c.client
		crs[i].errorHandler = defaultErrorHandler
	}
	return crs, nil
}
