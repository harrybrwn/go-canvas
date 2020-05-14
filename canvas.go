package canvas

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
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
	return getCourses(c.client, "/courses", asParams(opts))
}

// ActiveCourses returns a list of only the courses that are
// currently active
func (c *Canvas) ActiveCourses(options ...string) ([]*Course, error) {
	return getCourses(c.client, "/courses", &url.Values{
		"enrollment_state": {"active"},
		"include[]":        options,
	})
}

// CompletedCourses returns a list of only the courses that are
// not currently active and have been completed
func (c *Canvas) CompletedCourses(options ...string) ([]*Course, error) {
	return getCourses(c.client, "/courses", &url.Values{
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

// Accounts will list the accounts
func (c *Canvas) Accounts(opts ...Option) (accounts []Account, err error) {
	return accounts, getarr(c.client, &accounts, asParams(opts), "/accounts")
}

// CourseAccounts will make a call to the course accounts endpoint
func (c *Canvas) CourseAccounts(opts ...Option) (acts []Account, err error) {
	err = getarr(c.client, &acts, asParams(opts), "/course_accounts")
	if err != nil {
		return nil, err
	}
	for i := range acts {
		acts[i].cli = c.client
	}
	return
}

// Account is an account
type Account struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	UUID            string `json:"uuid"`
	ParentAccountID int    `json:"parent_account_id"`
	RootAccountID   int    `json:"root_account_id"`
	WorkflowState   string `json:"workflow_state"`
	DefaultTimeZone string `json:"default_time_zone"`
	IntegrationID   string `json:"integration_id"`
	SisAccountID    string `json:"sis_account_id"`
	SisImportID     int    `json:"sis_import_id"`
	LtiGUID         string `json:"lti_guid"`

	// Storage Quotas
	DefaultStorageQuotaMB      int `json:"default_storage_quota_mb"`
	DefaultUserStorageQuotaMB  int `json:"default_user_storage_quota_mb"`
	DefaultGroupStorageQuotaMB int `json:"default_group_storage_quota_mb"`

	Domain   string      `json:"domain"`
	Distance interface{} `json:"distance"`
	// Authentication Provider
	AuthProvider string `json:"authentication_provider"`

	cli doer
}

// Courses returns the account's list of courses
func (a *Account) Courses(opts ...Option) (courses []*Course, err error) {
	err = getarr(a.cli, &courses, asParams(opts), "/accounts/%d/courses", a.ID)
	if err != nil {
		return nil, err
	}
	for i := range courses {
		courses[i].client = a.cli
	}
	return
}

// SearchAccounts will search for canvas accounts.
// Options: name, domain, latitude, longitude
//
// 	c.SearchAccouts(Opt("name", "My School Name"))
func (c *Canvas) SearchAccounts(opts ...Option) (acts []Account, err error) {
	err = getarr(c.client, &acts, asParams(opts), "/accounts/search")
	if err != nil {
		return nil, err
	}
	for i := range acts {
		acts[i].cli = c.client
	}
	return
}

// Announcements will get the announcements
func (c *Canvas) Announcements(contexCodes []string, opts ...Option) (arr []DiscussionTopic, err error) {
	params := asParams(opts)
	params["context_codes"] = contexCodes
	return arr, getarr(c.client, &arr, params, "/announcements")
}

// DiscussionTopic is a discussion topic
type DiscussionTopic struct {
	ID                      int         `json:"id"`
	Title                   string      `json:"title"`
	Message                 string      `json:"message"`
	HTMLURL                 string      `json:"html_url"`
	PostedAt                time.Time   `json:"posted_at"`
	LastReplyAt             time.Time   `json:"last_reply_at"`
	RequireInitialPost      bool        `json:"require_initial_post"`
	UserCanSeePosts         bool        `json:"user_can_see_posts"`
	DiscussionSubentryCount int         `json:"discussion_subentry_count"`
	ReadState               string      `json:"read_state"`
	UnreadCount             int         `json:"unread_count"`
	Subscribed              bool        `json:"subscribed"`
	SubscriptionHold        string      `json:"subscription_hold"`
	AssignmentID            interface{} `json:"assignment_id"`
	DelayedPostAt           interface{} `json:"delayed_post_at"`
	Published               bool        `json:"published"`
	LockAt                  interface{} `json:"lock_at"`
	Locked                  bool        `json:"locked"`
	Pinned                  bool        `json:"pinned"`
	LockedForUser           bool        `json:"locked_for_user"`
	LockInfo                interface{} `json:"lock_info"`
	LockExplanation         string      `json:"lock_explanation"`
	UserName                string      `json:"user_name"`
	TopicChildren           []int       `json:"topic_children"`
	GroupTopicChildren      []struct {
		ID      int `json:"id"`
		GroupID int `json:"group_id"`
	} `json:"group_topic_children"`
	RootTopicID     interface{} `json:"root_topic_id"`
	PodcastURL      string      `json:"podcast_url"`
	DiscussionType  string      `json:"discussion_type"`
	GroupCategoryID interface{} `json:"group_category_id"`
	Attachments     interface{} `json:"attachments"`
	Permissions     struct {
		Attach bool `json:"attach"`
	} `json:"permissions"`
	AllowRating        bool `json:"allow_rating"`
	OnlyGradersCanRate bool `json:"only_graders_can_rate"`
	SortByRating       bool `json:"sort_by_rating"`
}

// CalendarEvents makes a call to get calendar events.
func (c *Canvas) CalendarEvents(opts ...Option) (cal []CalendarEvent, err error) {
	return cal, getarr(c.client, &cal, asParams(opts), "/calendar_events")
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

// Bookmarks will get the current user's bookmarks.
func (c *Canvas) Bookmarks(opts ...Option) (b []Bookmark, err error) {
	return b, getarr(c.client, &b, asParams(opts), "/users/self/bookmarks")
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

func getCourses(c doer, path string, vals encoder) (crs []*Course, err error) {
	err = getarr(c, &crs, vals, path)
	if err != nil {
		return nil, err
	}
	for i := range crs {
		crs[i].client = c
		crs[i].errorHandler = defaultErrorHandler
	}
	return crs, nil
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
