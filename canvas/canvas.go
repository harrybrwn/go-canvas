package canvas

import (
	"bytes"
	"encoding/json"
	"errors"
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
	return getUser(c.client, id, opts)
}

// CurrentUser get the currently logged in user.
func (c *Canvas) CurrentUser(opts ...Option) (*User, error) {
	return getUser(c.client, "self", opts)
}

// CurrentUserTodo will get the current user's todo's.
func (c *Canvas) CurrentUserTodo() error {
	panic("not implimented")
}

// CurrentAccount will get the current account.
func (c *Canvas) CurrentAccount() (*Account, error) {
	res := struct {
		*Account
		*AuthError
	}{nil, nil}
	if err := getjson(c.client, &res, "/accounts/self", nil); err != nil {
		return nil, err
	}
	if res.AuthError != nil {
		return nil, res.AuthError
	}
	return res.Account, nil
}

// Announcements will get the announcements
func (c *Canvas) Announcements(opts ...Option) error {
	res := struct {
		*AuthError
		Message string
	}{nil, ""}
	if err := getjson(c.client, &res, "/announcements", asParams(opts)); err != nil {
		return err
	}
	if res.AuthError != nil {
		return res.AuthError
	}
	if res.Message == "" {
		return errors.New(res.Message)
	}
	return nil
}

// CalendarEvents makes a call to get calendar events.
func (c *Canvas) CalendarEvents(opts ...Option) ([]CalendarEvent, error) {
	cal := []CalendarEvent{}
	if err := getjson(c.client, &cal, "/calendar_events", asParams(opts)); err != nil {
		return nil, err
	}
	return cal, nil
}

// CalendarEvent is a calendar event
type CalendarEvent struct {
	ID                         int         `json:"id"`
	Title                      string      `json:"title"`
	StartAt                    string      `json:"start_at"`
	EndAt                      string      `json:"end_at"`
	Description                string      `json:"description"`
	LocationName               string      `json:"location_name"`
	LocationAddress            string      `json:"location_address"`
	ContextCode                string      `json:"context_code"`
	EffectiveContextCode       interface{} `json:"effective_context_code"`
	AllContextCodes            string      `json:"all_context_codes"`
	WorkflowState              string      `json:"workflow_state"`
	Hidden                     bool        `json:"hidden"`
	ParentEventID              interface{} `json:"parent_event_id"`
	ChildEventsCount           int         `json:"child_events_count"`
	ChildEvents                interface{} `json:"child_events"`
	URL                        string      `json:"url"`
	HTMLURL                    string      `json:"html_url"`
	AllDayDate                 string      `json:"all_day_date"`
	AllDay                     bool        `json:"all_day"`
	CreatedAt                  string      `json:"created_at"`
	UpdatedAt                  string      `json:"updated_at"`
	AppointmentGroupID         interface{} `json:"appointment_group_id"`
	AppointmentGroupURL        interface{} `json:"appointment_group_url"`
	OwnReservation             bool        `json:"own_reservation"`
	ReserveURL                 interface{} `json:"reserve_url"`
	Reserved                   bool        `json:"reserved"`
	ParticipantType            string      `json:"participant_type"`
	ParticipantsPerAppointment interface{} `json:"participants_per_appointment"`
	AvailableSlots             interface{} `json:"available_slots"`
	User                       interface{} `json:"user"`
	Group                      interface{} `json:"group"`
}

// Accounts will list the accounts
func (c *Canvas) Accounts(opts ...Option) ([]Account, error) {
	accounts := []Account{}
	if err := getjson(c.client, &accounts, "/accounts", asParams(opts)); err != nil {
		return nil, err
	}
	return accounts, nil
}

// CourseAccounts will make a call to the course accounts endpoint
func (c *Canvas) CourseAccounts(opts ...Option) ([]Account, error) {
	accounts := []Account{}
	if err := getjson(c.client, &accounts, "/course_accounts", asParams(opts)); err != nil {
		return nil, err
	}
	return accounts, nil
}

// Account is an account
type Account struct{}

// Bookmarks will get the current user's bookmarks.
func (c *Canvas) Bookmarks(opts ...Option) ([]Bookmark, error) {
	return getBookmarks(c.client, "self", opts)
}

// CreateBookmark will take a bookmark and send it to canvas.
func (c *Canvas) CreateBookmark(b *Bookmark) error {
	return createBookmark(c.client, "self", b)
}

// Bookmark is a bookmark object.
type Bookmark struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	Position int    `json:"position"`
	Data     struct {
		ActiveTab int `json:"active_tab"`
	} `json:"data"`
}

// pathVar is an interface{} because internally, either "self" or some integer id
// will be passed to be used as an api path parameter.
func getUser(c doer, pathVar interface{}, opts []Option) (*User, error) {
	res := struct {
		*User
		*AuthError
	}{&User{client: c}, nil}

	if err := getjson(
		c, &res,
		fmt.Sprintf("users/%v", pathVar),
		asParams(opts),
	); err != nil {
		return nil, err
	}
	if res.AuthError != nil {
		return nil, res.AuthError
	}
	return res.User, nil
}

func (c *Canvas) getCourses(vals encoder) ([]*Course, error) {
	crs := make([]*Course, 0)
	resp, err := get(c.client, "/courses", vals)
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
		e := &AuthError{}
		err = json.Unmarshal(b.Bytes(), e)
		return nil, errpair(e, err) // return any and all non-nil errors
	}

	for i := range crs {
		crs[i].client = c.client
		crs[i].errorHandler = defaultErrorHandler
	}
	return crs, nil
}

func getBookmarks(d doer, id interface{}, opts []Option) ([]Bookmark, error) {
	resp, err := get(d, fmt.Sprintf("users/%v/bookmarks", id), asParams(opts))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var (
		b         bytes.Buffer
		bookmarks []Bookmark
	)
	if _, err = b.ReadFrom(resp.Body); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(b.Bytes(), &bookmarks); err != nil {
		e := &AuthError{}
		return nil, errpair(json.Unmarshal(b.Bytes(), e), e)
	}
	return bookmarks, nil
}

func createBookmark(d doer, id interface{}, b *Bookmark) error {
	resp, err := post(d, fmt.Sprintf("/users/%v/bookmarks", id), params{
		"name":     {b.Name},
		"url":      {b.URL},
		"position": {fmt.Sprintf("%d", b.Position)},
		"data":     {},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	e := &AuthError{}
	return errpair(json.NewDecoder(resp.Body).Decode(e), e)
}
