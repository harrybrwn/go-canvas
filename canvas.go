package canvas

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

var (
	// DefaultHost is the default url host for the canvas api.
	DefaultHost = "canvas.instructure.com"

	// ConcurrentErrorHandler is the error handling callback for
	// handling errors in tricky goroutines.
	ConcurrentErrorHandler func(error, chan int) = defaultErrorHandler

	// DefaultUserAgent is the default user agent used to make requests.
	DefaultUserAgent = "go-canvas"

	// DefaultCanvas is the default canvas object
	defaultCanvas *Canvas
)

func init() {
	token := os.Getenv("CANVAS_TOKEN")
	defaultCanvas = New(token)
}

// SetToken will set the package level canvas object token.
func SetToken(token string) {
	defaultCanvas = New(token)
}

// SetHost will set the package level host.
func SetHost(host string) error {
	return defaultCanvas.SetHost(host)
}

// New will create a Canvas struct from an api token.
// New uses the default host.
func New(token string) *Canvas {
	return WithHost(token, DefaultHost)
}

// WithHost will create a canvas object that uses a
// different hostname.
func WithHost(token, host string) *Canvas {
	c := &Canvas{client: &http.Client{}}
	authorize(c.client, token, host)
	return c
}

// Canvas is the main api controller.
type Canvas struct {
	client *http.Client
}

// SetHost will set the host for the canvas requestor.
func (c *Canvas) SetHost(host string) error {
	auth, ok := c.client.Transport.(*auth)
	if !ok {
		return errors.New("could not set canvas host")
	}
	auth.host = host
	return nil
}

// Courses lists all of the courses associated
// with that canvas object.
func (c *Canvas) Courses(opts ...Option) ([]*Course, error) {
	return getCourses(c.client, "/courses", asParams(opts))
}

// Courses lists all of the courses associated
// with that canvas object.
func Courses(opts ...Option) ([]*Course, error) {
	return defaultCanvas.Courses(opts...)
}

// GetCourse will get a course given a course id.
func (c *Canvas) GetCourse(id int, opts ...Option) (*Course, error) {
	course := &Course{client: c.client}
	return course, getjson(c.client, &course, asParams(opts), "/courses/%d", id)
}

// GetCourse will get a course given a course id.
func GetCourse(id int, opts ...Option) (*Course, error) {
	return defaultCanvas.GetCourse(id, opts...)
}

// ActiveCourses returns a list of only the courses that are
// currently active
func (c *Canvas) ActiveCourses(opts ...Option) ([]*Course, error) {
	p := params{"enrollment_state": {"active"}}
	p.Add(opts...)
	return getCourses(c.client, "/courses", p)
}

// ActiveCourses returns a list of only the courses that are
// currently active
func ActiveCourses(opts ...Option) ([]*Course, error) {
	return defaultCanvas.ActiveCourses(opts...)
}

// CompletedCourses returns a list of only the courses that are
// not currently active and have been completed
func (c *Canvas) CompletedCourses(opts ...Option) ([]*Course, error) {
	p := params{"enrollment_state": {"completed"}}
	p.Add(opts...)
	return getCourses(c.client, "/courses", p)
}

// CompletedCourses returns a list of only the courses that are
// not currently active and have been completed
func CompletedCourses(opts ...Option) ([]*Course, error) {
	return defaultCanvas.CompletedCourses(opts...)
}

// GetUser will return a user object given that user's ID.
func (c *Canvas) GetUser(id int, opts ...Option) (*User, error) {
	return getUser(c.client, id, opts)
}

// GetUser will return a user object given that user's ID.
func GetUser(id int, opts ...Option) (*User, error) {
	return defaultCanvas.GetUser(id, opts...)
}

// CurrentUser get the currently logged in user.
func (c *Canvas) CurrentUser(opts ...Option) (*User, error) {
	return getUser(c.client, "self", opts)
}

// CurrentUser get the currently logged in user.
func CurrentUser(opts ...Option) (*User, error) {
	return defaultCanvas.CurrentUser(opts...)
}

// Todos will get the current user's todo's.
func (c *Canvas) Todos() error {
	panic("not implimented")
}

// Todos will get the current user's todo's.
func Todos() error {
	return defaultCanvas.Todos()
}

// CurrentAccount will get the current account.
func (c *Canvas) CurrentAccount() (a *Account, err error) {
	a = &Account{cli: c.client}
	return a, getjson(c.client, a, nil, "/accounts/self")
}

// CurrentAccount will get the current account.
func CurrentAccount() (a *Account, err error) {
	return defaultCanvas.CurrentAccount()
}

// Accounts will list the accounts
func (c *Canvas) Accounts(opts ...Option) ([]Account, error) {
	return getAccounts(c.client, "/accounts", opts)
}

// Account will list a single under an account
func (c *Canvas) Account(accountId int, opts ...Option) (*Account, error) {
	return getAccount(c.client, fmt.Sprintf("/accounts/%d", accountId), opts)
}

// SubAccounts will list the sub_accounts under an account
func (c *Canvas) SubAccounts(accountId int, opts ...Option) ([]Account, error) {
	return getAccounts(c.client, fmt.Sprintf("/accounts/%d/sub_accounts", accountId), opts)
}

// Accounts will list the accounts
func Accounts(opts ...Option) ([]Account, error) {
	return defaultCanvas.Accounts()
}

// CourseAccounts will make a call to the course accounts endpoint
func (c *Canvas) CourseAccounts(opts ...Option) ([]Account, error) {
	return getAccounts(c.client, "/course_accounts", opts)
}

// CourseAccounts will make a call to the course accounts endpoint
func CourseAccounts(opts ...Option) ([]Account, error) {
	return defaultCanvas.CourseAccounts()
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
	return getCourses(a.cli, fmt.Sprintf("/accounts/%d/courses", a.ID), asParams(opts))
}

// SearchAccounts will search for canvas accounts.
// Options: name, domain, latitude, longitude
//
// 	c.SearchAccouts(Opt("name", "My School Name"))
func (c *Canvas) SearchAccounts(opts ...Option) ([]Account, error) {
	return getAccounts(c.client, "accounts/search", opts)
}

// SearchAccounts will search for canvas accounts.
// Options: name, domain, latitude, longitude
//
// 	c.SearchAccouts(Opt("name", "My School Name"))
func SearchAccounts(opts ...Option) ([]Account, error) {
	return defaultCanvas.SearchAccounts(opts...)
}

// Announcements will get the announcements
func (c *Canvas) Announcements(contextCodes []string, opts ...Option) (arr []DiscussionTopic, err error) {
	p := params{"context_codes": contextCodes}
	p.Add(opts...)
	return arr, getjson(c.client, &arr, p, "/announcements")
}

// Announcements will get the announcements
func Announcements(contextCodes []string, opts ...Option) ([]DiscussionTopic, error) {
	return defaultCanvas.Announcements(contextCodes, opts...)
}

// CalendarEvents makes a call to get calendar events.
func (c *Canvas) CalendarEvents(opts ...Option) (cal []CalendarEvent, err error) {
	return cal, getjson(c.client, &cal, asParams(opts), "/calendar_events")
}

// CalendarEvents makes a call to get calendar events.
func CalendarEvents(opts ...Option) ([]CalendarEvent, error) {
	return defaultCanvas.CalendarEvents(opts...)
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

// CalendarEvent is a calendar event
type CalendarEvent struct {
	// ID                         int         `json:"id"`
	ID                         string      `json:"id"`
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
	ReserveURL                 string      `json:"reserve_url"`
	Reserved                   bool        `json:"reserved"`
	ParticipantType            string      `json:"participant_type"`
	ParticipantsPerAppointment interface{} `json:"participants_per_appointment"`
	AvailableSlots             interface{} `json:"available_slots"`
	User                       *User       `json:"user"`
	Group                      interface{} `json:"group"`
}

// Conversations returns a list of conversations
func (c *Canvas) Conversations(opts ...Option) (conversations []Conversation, err error) {
	return conversations, getjson(c.client, &conversations, asParams(opts), "/conversations")
}

// Conversations returns a list of conversations
func Conversations(opts ...Option) ([]Conversation, error) {
	return defaultCanvas.Conversations(opts...)
}

// Conversation is a conversation.
type Conversation struct {
	ID               int         `json:"id"`
	Subject          string      `json:"subject"`
	WorkflowState    string      `json:"workflow_state"`
	LastMessage      string      `json:"last_message"`
	StartAt          time.Time   `json:"start_at"`
	MessageCount     int         `json:"message_count"`
	Subscribed       bool        `json:"subscribed"`
	Private          bool        `json:"private"`
	Starred          bool        `json:"starred"`
	Properties       interface{} `json:"properties"`
	Audience         interface{} `json:"audience"`
	AudienceContexts interface{} `json:"audience_contexts"`
	AvatarURL        string      `json:"avatar_url"`
	Participants     interface{} `json:"participants"`
	Visible          bool        `json:"visible"`
	ContextName      string      `json:"context_name"`
}

// Bookmarks will get the current user's bookmarks.
func (c *Canvas) Bookmarks(opts ...Option) (b []Bookmark, err error) {
	return b, getjson(c.client, &b, asParams(opts), "/users/self/bookmarks")
}

// CreateBookmark will take a bookmark and send it to canvas.
func (c *Canvas) CreateBookmark(b *Bookmark) error {
	return createBookmark(c.client, "self", b)
}

// Bookmarks will get the current user's bookmarks.
func Bookmarks(opts ...Option) ([]Bookmark, error) {
	return defaultCanvas.Bookmarks(opts...)
}

// CreateBookmark will take a bookmark and send it to canvas.
func CreateBookmark(b *Bookmark) error {
	return defaultCanvas.CreateBookmark(b)
}

// DeleteBookmark will delete a bookmark
func (c *Canvas) DeleteBookmark(b *Bookmark) error {
	return deleteBookmark(c.client, "self", b.ID)
}

// DeleteBookmark will delete a bookmark
func DeleteBookmark(b *Bookmark) error {
	return defaultCanvas.DeleteBookmark(b)
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
func getUser(c doer, pathVar interface{}, opts []Option) (u *User, err error) {
	u = &User{client: c}
	if err = getjson(c, u, asParams(opts), "users/%v", pathVar); err != nil {
		return nil, err
	}
	return u, nil
}

func getCourses(c doer, path string, vals encoder) (crs []*Course, err error) {
	err = getjson(c, &crs, vals, path)
	if err != nil {
		return nil, err
	}
	for i := range crs {
		crs[i].client = c
		crs[i].errorHandler = ConcurrentErrorHandler
	}
	return crs, nil
}

func createBookmark(d doer, id interface{}, b *Bookmark) error {
	p := params{
		"name":     {b.Name},
		"position": {fmt.Sprintf("%d", b.Position)},
	}
	if b.URL != "" {
		p["url"] = []string{b.URL}
	}
	resp, err := post(d, fmt.Sprintf("/users/%v/bookmarks", id), p)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return err
}

func deleteBookmark(d doer, pathvar interface{}, id int) error {
	req := newreq("DELETE", fmt.Sprintf("/users/%v/bookmarks/%d", pathvar, id), "")
	if _, err := do(d, req); err != nil {
		return err
	}
	return nil
}

func getAccounts(d doer, path string, opts []Option) (accts []Account, err error) {
	err = getjson(d, &accts, asParams(opts), path)
	if err != nil {
		return nil, err
	}
	for i := range accts {
		accts[i].cli = d
	}
	return
}


func getAccount(d doer, path string, opts []Option) (acct *Account, err error) {
	acct = &Account{cli: d}
	err = getjson(d, &acct, asParams(opts), path)
	if err != nil {
		return acct, err
	}
	return
}