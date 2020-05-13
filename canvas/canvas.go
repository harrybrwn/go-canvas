package canvas

import (
	"fmt"
	"net/url"
)

// FromToken will create a Canvas struct from an api token
func FromToken(token string) *Canvas {
	return &Canvas{
		client: newclient(token),
	}
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
	u := &User{client: c.client}
	return u, c.client.getjson(u, fmt.Sprintf("users/%d", id), asParams(opts))
}

// CurrentUser get the currently logged in user.
func (c *Canvas) CurrentUser(opts ...Option) (*User, error) {
	u := &User{client: c.client}
	return u, getjson(c.client, u, "users/self", asParams(opts))
}

// CurrentUserTodo will get the current user's todo's.
func (c *Canvas) CurrentUserTodo() error {
	panic("not implimented")
}

func (c *Canvas) getCourses(vals encoder) ([]*Course, error) {
	crs := make([]*Course, 0)
	if err := c.client.getjson(&crs, "courses", vals); err != nil {
		return nil, err
	}
	for i := range crs {
		crs[i].client = c.client
		crs[i].errorHandler = defaultErrorHandler
	}
	return crs, nil
}
