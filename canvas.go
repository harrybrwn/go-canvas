// Package canvas is an API wrapper for Instructure's Canvas API written in Go.
//
// For the official Canvas API documentation, see https://canvas.instructure.com/doc/api/all_resources.html
package canvas

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/harrybrwn/go-querystring/query"
)

var (
	// DefaultHost is the default url host for the canvas api.
	DefaultHost = "canvas.instructure.com"

	// ConcurrentErrorHandler is the error handling callback for
	// handling errors in tricky goroutines.
	//
	// If you do not want to stop all concurrent goroutines, this
	// handler should return an non-nil error. If this handler returns
	// nil then all goroutines will continue if they can.
	// This function panics by default.
	ConcurrentErrorHandler func(error) error = defaultErrorHandler

	// DefaultUserAgent is the default user agent used to make requests.
	DefaultUserAgent = "go-canvas v0.1"

	// DefaultCanvas is the default canvas object
	ca *Canvas
)

func init() {
	token := os.Getenv("CANVAS_TOKEN")
	SetToken(token)
}

// SetToken will set the package level canvas object token.
func SetToken(token string) {
	ca = New(token)
}

// SetHost will set the package level host.
func SetHost(host string) error { return ca.SetHost(host) }

// New will create a Canvas struct from an api token.
// New uses the default host.
func New(token string) *Canvas {
	return WithHost(token, DefaultHost)
}

// WithHost will create a canvas object that uses a
// different hostname.
func WithHost(token, host string) *Canvas {
	c := http.Client{}
	authorize(&c, token, host)
	return &Canvas{&client{Client: c, host: host}}
}

// Canvas is the main api entry point.
type Canvas struct {
	client doer
	// client *client
}

// SetHost will set the host for the canvas requestor.
func (c *Canvas) SetHost(host string) error {
	auth, ok := c.client.(*client).Transport.(*auth)
	if !ok {
		return errors.New("could not set canvas host")
	}
	auth.host = host
	return nil
}

// Courses lists all of the courses associated
// with that canvas object.
//
// https://canvas.instructure.com/doc/api/courses.html#method.courses.index
func Courses(opts ...Option) ([]*Course, error) { return ca.Courses(opts...) }

// Courses lists all of the courses associated
// with that canvas object.
//
// https://canvas.instructure.com/doc/api/courses.html#method.courses.index
func (c *Canvas) Courses(opts ...Option) ([]*Course, error) {
	return getCourses(c.client, "/courses", optEnc(opts))
}

func getCourses(c doer, path string, opts optEnc) (crs []*Course, err error) {
	ch := make(chan *Course)
	pager := newPaginatedList(
		c, path, func(r io.Reader) error {
			list := make([]*Course, 0)
			if err := json.NewDecoder(r).Decode(&list); err != nil {
				return err
			}
			for _, course := range list {
				course.client = c
				course.errorHandler = ConcurrentErrorHandler
				ch <- course
			}
			return nil
		}, opts,
	)
	errs := pager.start()
	for {
		select {
		case course := <-ch:
			crs = append(crs, course)
		case err := <-errs:
			return crs, err
		}
	}
}

// CoursesChan returns a channel of courses
func CoursesChan(opts ...Option) <-chan *Course {
	return ca.CoursesChan(opts...)
}

// CoursesChan returns a channel of courses
func (c *Canvas) CoursesChan(opts ...Option) <-chan *Course {
	ch := make(courseChan)
	pager := newPaginatedList(
		c.client, "/courses", func(r io.Reader) error {
			list := make([]*Course, 0)
			if err := json.NewDecoder(r).Decode(&list); err != nil {
				return err
			}
			for _, course := range list {
				course.client = c.client
				course.errorHandler = ConcurrentErrorHandler
				ch <- course
			}
			return nil
		}, opts)
	go handleErrs(pager.start(), ch, ConcurrentErrorHandler)
	return ch
}

// GetCourse will get a course given a course id.
//
// https://canvas.instructure.com/doc/api/courses.html#method.courses.show
func GetCourse(id int, opts ...Option) (*Course, error) { return ca.GetCourse(id, opts...) }

// GetCourse will get a course given a course id.
//
// https://canvas.instructure.com/doc/api/courses.html#method.courses.show
func (c *Canvas) GetCourse(id int, opts ...Option) (*Course, error) {
	course := &Course{client: c.client, errorHandler: ConcurrentErrorHandler}
	return course, getjson(c.client, &course, optEnc(opts), "/courses/%d", id)
}

// GetUser will return a user object given that user's ID.
func (c *Canvas) GetUser(id int, opts ...Option) (*User, error) {
	return getUser(c.client, id, opts)
}

// GetUser will return a user object given that user's ID.
func GetUser(id int, opts ...Option) (*User, error) { return ca.GetUser(id, opts...) }

// CurrentUser get the currently logged in user.
func (c *Canvas) CurrentUser(opts ...Option) (*User, error) {
	return getUser(c.client, "self", opts)
}

// CurrentUser get the currently logged in user.
func CurrentUser(opts ...Option) (*User, error) { return ca.CurrentUser(opts...) }

// Todos will get the current user's todo's.
func (c *Canvas) Todos() ([]TODO, error) {
	todos := make([]TODO, 0)
	return todos, getjson(c.client, &todos, url.Values{"per_page": {"100"}}, "/users/self/todo")
}

// Todos will get the current user's todo's.
func Todos() ([]TODO, error) { return ca.Todos() }

// TODO is a to-do struct
type TODO struct {
	Type              string      `json:"type"`
	Assignement       *Assignment `json:"assignment"`
	Ignore            string      `json:"ignore"`
	IgnorePerminantly string      `json:"ignore_perminantly"`
	HTMLURL           string      `json:"html_url"`
	NeedsGradingCount int         `json:"needs_grading_count"`
	ContextType       string      `json:"context_type"`
	ContextID         int         `json:"context_id"`
	CourseID          int         `json:"course_id"`
	GroupID           interface{} `json:"group_id"`
}

// NewFile will make a new file object. This will not
// send any data to canvas.
func NewFile(filename string) *File { return ca.NewFile(filename) }

// NewFile will make a new file object. This will not
// send any data to canvas.
func (c *Canvas) NewFile(filename string) *File {
	return &File{Filename: filename, client: c.client}
}

// NewFolder will make a new folder object. This will not
// send any data to canvas.
func NewFolder(foldername string) *Folder { return ca.NewFolder(foldername) }

// NewFolder will make a new folder object. This will not
// send any data to canvas.
func (c *Canvas) NewFolder(foldername string) *Folder {
	return &Folder{Foldername: foldername, client: c.client}
}

// GetFile will get a file by the id.
func (c *Canvas) GetFile(id int, opts ...Option) (*File, error) {
	return getUserFile(c.client, id, "self", opts)
}

// GetFile will get a file by the id.
func GetFile(id int, opts ...Option) (*File, error) { return ca.GetFile(id, opts...) }

// Files will return a channel of all the default user's files.
// https://canvas.instructure.com/doc/api/files.html#method.files.api_index
func (c *Canvas) Files(opts ...Option) <-chan *File {
	return filesChannel(c.client, "/users/self/files", ConcurrentErrorHandler, opts, nil)
}

// ListFiles will return a slice of the current user's files.
func (c *Canvas) ListFiles(opts ...Option) ([]*File, error) {
	return listFiles(c.client, "/users/self/files", nil, opts)
}

// ListFiles will return a slice of the current user's files.
func ListFiles(opts ...Option) ([]*File, error) { return ca.ListFiles(opts...) }

// Files will return a channel of all the default user's files.
// https://canvas.instructure.com/doc/api/files.html#method.files.api_index
func Files(opts ...Option) <-chan *File { return ca.Files(opts...) }

// Folders returns a channel of folders for the current user.
func (c *Canvas) Folders(opts ...Option) <-chan *Folder {
	return foldersChannel(
		c.client,
		"/users/self/folders",
		ConcurrentErrorHandler,
		opts, nil,
	)
}

// Folders returns a channel of folders for the current user.
func Folders(opts ...Option) <-chan *Folder { return ca.Folders(opts...) }

// ListFolders will return a slice of the current user's folders
func (c *Canvas) ListFolders(opts ...Option) ([]*Folder, error) {
	return listFolders(c.client, "/users/self/folders", nil, opts)
}

// ListFolders will return a slice of the current user's folders
func ListFolders(opts ...Option) ([]*Folder, error) { return ca.ListFolders(opts...) }

// FolderPath will get a list of folders in the path given.
func (c *Canvas) FolderPath(folderpath string) ([]*Folder, error) {
	folderpath = path.Join("/users/self/folders/by_path", folderpath)
	return folderList(c.client, folderpath)
}

// FolderPath will get a list of folders in the path given.
func FolderPath(path string) ([]*Folder, error) { return ca.FolderPath(path) }

// Root will get the current user's root folder
func (c *Canvas) Root(opts ...Option) (*Folder, error) {
	f := &Folder{client: c.client}
	return f, getjson(c.client, f, optEnc(opts), "/users/self/folders/root")
}

// Root will get the current user's root folder
func Root(opts ...Option) (*Folder, error) {
	return ca.Root(opts...)
}

// CreateFolder will create a new folder.
func (c *Canvas) CreateFolder(path string, opts ...Option) (*Folder, error) {
	dir, name := filepath.Split(path)
	return createFolder(c.client, dir, name, opts, "/users/self/folders")
}

// CreateFolder will create a new folder.
func CreateFolder(path string, opts ...Option) (*Folder, error) {
	return ca.CreateFolder(path, opts...)
}

// UploadFile uploads a file to the current user's files.
func (c *Canvas) UploadFile(filename string, r io.Reader, opts ...Option) (*File, error) {
	return uploadFile(
		c.client, r, "/users/self/files",
		newFileUploadParams(filename, opts),
	)
}

// UploadFile uploads a file to the current user's files.
func UploadFile(filename string, r io.Reader, opts ...Option) (*File, error) {
	return ca.UploadFile(filename, r, opts...)
}

// CurrentAccount will get the current account.
func (c *Canvas) CurrentAccount() (a *Account, err error) {
	a = &Account{cli: c.client}
	return a, getjson(c.client, a, nil, "/accounts/self")
}

// CurrentAccount will get the current account.
func CurrentAccount() (a *Account, err error) { return ca.CurrentAccount() }

// Accounts will list the accounts
func (c *Canvas) Accounts(opts ...Option) ([]Account, error) {
	return getAccounts(c.client, "/accounts", opts)
}

// Accounts will list the accounts
func Accounts(opts ...Option) ([]Account, error) { return ca.Accounts() }

// CourseAccounts will make a call to the course accounts endpoint
func (c *Canvas) CourseAccounts(opts ...Option) ([]Account, error) {
	return getAccounts(c.client, "/course_accounts", opts)
}

// CourseAccounts will make a call to the course accounts endpoint
func CourseAccounts(opts ...Option) ([]Account, error) { return ca.CourseAccounts() }

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
	return getCourses(a.cli, fmt.Sprintf("/accounts/%d/courses", a.ID), optEnc(opts))
}

// SearchAccounts will search for canvas accounts.
// Options: name, domain, latitude, longitude
func (c *Canvas) SearchAccounts(term string, opts ...Option) ([]Account, error) {
	opts = append(opts, Opt("name", term))
	return getAccounts(c.client, "accounts/search", opts)
}

// SearchAccounts will search for canvas accounts.
// Options: name, domain, latitude, longitude
func SearchAccounts(term string, opts ...Option) ([]Account, error) {
	return ca.SearchAccounts(term, opts...)
}

// Announcements will get the announcements
// https://canvas.instructure.com/doc/api/all_resources.html#method.announcements_api.index
func (c *Canvas) Announcements(
	contextCodes []string,
	opts ...Option,
) (arr []*DiscussionTopic, err error) {
	opts = append(opts, Opt("context_codes[]", contextCodes))
	ch := make(chan *DiscussionTopic)
	pager := newPaginatedList(
		c.client, "/announcements",
		sendDiscussionTopicFunc(ch), opts)
	arr = make([]*DiscussionTopic, 0)
	errs := pager.start()
	for {
		select {
		case an := <-ch:
			arr = append(arr, an)
		case err := <-errs:
			return arr, err
		}
	}
}

// Announcements will get the announcements
func Announcements(
	contextCodes []string,
	opts ...Option,
) ([]*DiscussionTopic, error) {
	return ca.Announcements(contextCodes, opts...)
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
func (c *Canvas) CalendarEvents(opts ...Option) (cal []*CalendarEvent, err error) {
	ch := make(chan *CalendarEvent)
	pager := newPaginatedList(c.client, "/calendar_events", func(r io.Reader) error {
		evs := make([]*CalendarEvent, 0)
		if err := json.NewDecoder(r).Decode(&evs); err != nil {
			return err
		}
		for _, e := range evs {
			ch <- e
		}
		return nil
	}, opts)
	errs := pager.start()
	events := make([]*CalendarEvent, 0)
	for {
		select {
		case event := <-ch:
			events = append(events, event)
		case err := <-errs:
			return events, err
		}
	}
}

// CalendarEvents makes a call to get calendar events.
func CalendarEvents(opts ...Option) ([]*CalendarEvent, error) {
	return ca.CalendarEvents(opts...)
}

type calendarEventOptions struct {
	CalendarEvent `url:"calendar_event"`
}

// CreateCalendarEvent will send a calendar event to canvas to be created.
// https://canvas.instructure.com/doc/api/all_resources.html#method.calendar_events_api.create
func (c *Canvas) CreateCalendarEvent(event *CalendarEvent) (*CalendarEvent, error) {
	// TODO: figure out how to send theses fields:
	// 	- calendar_event[child_event_data][X][start_at]
	// 	- calendar_event[duplicate][count]
	// 	- calendar_event[duplicate][interval]
	// 	- calendar_event[duplicate][frequency]
	// see https://canvas.instructure.com/doc/api/all_resources.html#method.calendar_events_api.create
	q, err := query.Values(&calendarEventOptions{*event})
	if err != nil {
		return nil, err
	}
	resp, err := post(c.client, "/calendar_events", q)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	cal := &CalendarEvent{}
	return cal, json.NewDecoder(resp.Body).Decode(cal)
}

// CreateCalendarEvent will send a calendar event to canvas to be created.
// https://canvas.instructure.com/doc/api/all_resources.html#method.calendar_events_api.create
func CreateCalendarEvent(event *CalendarEvent) (*CalendarEvent, error) {
	return ca.CreateCalendarEvent(event)
}

// UpdateCalendarEvent will update a calendar event. This operation will change
// event given as an argument.
// https://canvas.instructure.com/doc/api/all_resources.html#method.calendar_events_api.update
func (c *Canvas) UpdateCalendarEvent(event *CalendarEvent) error {
	q, err := query.Values(&calendarEventOptions{*event})
	if err != nil {
		return err
	}
	resp, err := put(c.client, fmt.Sprintf("/calendar_events/%d", event.ID), q)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(event)
}

// UpdateCalendarEvent will update a calendar event. This operation will change
// event given as an argument.
// https://canvas.instructure.com/doc/api/all_resources.html#method.calendar_events_api.update
func UpdateCalendarEvent(event *CalendarEvent) error {
	return ca.UpdateCalendarEvent(event)
}

// DeleteCalendarEventByID will delete a calendar event given its ID.
// This operation returns the calendar event that was deleted.
func (c *Canvas) DeleteCalendarEventByID(id int, opts ...Option) (*CalendarEvent, error) {
	resp, err := delete(c.client, fmt.Sprintf("/calendar_events/%d", id), optEnc(opts))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	e := &CalendarEvent{}
	return e, json.NewDecoder(resp.Body).Decode(e)
}

// DeleteCalendarEventByID will delete a calendar event given its ID.
// This operation returns the calendar event that was deleted.
func DeleteCalendarEventByID(id int, opts ...Option) (*CalendarEvent, error) {
	return ca.DeleteCalendarEventByID(id, opts...)
}

// DeleteCalendarEvent will delete the calendar event and
// return the calendar event deleted.
func (c *Canvas) DeleteCalendarEvent(e *CalendarEvent) (*CalendarEvent, error) {
	return c.DeleteCalendarEventByID(e.ID)
}

// DeleteCalendarEvent will delete the calendar event and
// return the calendar event deleted.
func DeleteCalendarEvent(e *CalendarEvent) (*CalendarEvent, error) {
	return ca.DeleteCalendarEventByID(e.ID)
}

// CalendarEvent is a calendar event
type CalendarEvent struct {
	ID                         int         `json:"id" url:"-"`
	Title                      string      `json:"title" url:"title,omitempty"`
	ContextCode                string      `json:"context_code" url:"context_code,omitempty"`
	StartAt                    time.Time   `json:"start_at" url:"start_at,omitempty"`
	EndAt                      time.Time   `json:"end_at" url:"end_at,omitempty"`
	CreatedAt                  time.Time   `json:"created_at" url:"-"`
	UpdatedAt                  time.Time   `json:"updated_at" url:"-"`
	Description                string      `json:"description" url:"description,omitempty"`
	LocationName               string      `json:"location_name" url:"location_name,omitempty"`
	LocationAddress            string      `json:"location_address" url:"location_address,omitempty"`
	EffectiveContextCode       interface{} `json:"effective_context_code" url:"effective_context_code,omitempty"`
	AllDay                     bool        `json:"all_day" url:"all_day,omitempty"`
	AllContextCodes            string      `json:"all_context_codes" url:"-"`
	WorkflowState              string      `json:"workflow_state" url:"-"`
	Hidden                     bool        `json:"hidden" url:"-"`
	ParentEventID              interface{} `json:"parent_event_id" url:"-"`
	ChildEventsCount           int         `json:"child_events_count" url:"-"`
	ChildEvents                interface{} `json:"child_events" url:"-"`
	URL                        string      `json:"url" url:"-"`
	HTMLURL                    string      `json:"html_url" url:"-"`
	AllDayDate                 string      `json:"all_day_date" url:"-"`
	AppointmentGroupID         interface{} `json:"appointment_group_id" url:"-"`
	AppointmentGroupURL        string      `json:"appointment_group_url" url:"-"`
	OwnReservation             bool        `json:"own_reservation" url:"-"`
	ReserveURL                 string      `json:"reserve_url" url:"-"`
	Reserved                   bool        `json:"reserved" url:"-"`
	ParticipantType            string      `json:"participant_type" url:"-"`
	ParticipantsPerAppointment interface{} `json:"participants_per_appointment" url:"-"`
	AvailableSlots             interface{} `json:"available_slots" url:"-"`
	User                       *User       `json:"user" url:"-"`
	Group                      interface{} `json:"group" url:"-"`
}

// Conversations returns a list of conversations
func (c *Canvas) Conversations(opts ...Option) (conversations []Conversation, err error) {
	return conversations, getjson(c.client, &conversations, optEnc(opts), "/conversations")
}

// Conversations returns a list of conversations
func Conversations(opts ...Option) ([]Conversation, error) {
	return ca.Conversations(opts...)
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
	return b, getjson(c.client, &b, optEnc(opts), "/users/self/bookmarks")
}

// CreateBookmark will take a bookmark and send it to canvas.
func (c *Canvas) CreateBookmark(b *Bookmark) error {
	return createBookmark(c.client, "self", b)
}

// Bookmarks will get the current user's bookmarks.
func Bookmarks(opts ...Option) ([]Bookmark, error) { return ca.Bookmarks(opts...) }

// CreateBookmark will take a bookmark and send it to canvas.
func CreateBookmark(b *Bookmark) error { return ca.CreateBookmark(b) }

// DeleteBookmark will delete a bookmark
func (c *Canvas) DeleteBookmark(b *Bookmark) error {
	return deleteBookmark(c.client, "self", b.ID)
}

// DeleteBookmark will delete a bookmark
func DeleteBookmark(b *Bookmark) error { return ca.DeleteBookmark(b) }

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
	if err = getjson(c, u, optEnc(opts), "users/%v", pathVar); err != nil {
		return nil, err
	}
	return u, nil
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
	return resp.Body.Close()
}

func deleteBookmark(d doer, pathvar interface{}, id int) error {
	_, err := delete(d, fmt.Sprintf("/users/%v/bookmarks/%d", pathvar, id), nil)
	return err
}

func getAccounts(d doer, path string, opts []Option) (accts []Account, err error) {
	err = getjson(d, &accts, optEnc(opts), path)
	if err != nil {
		return nil, err
	}
	for i := range accts {
		accts[i].cli = d
	}
	return
}

func sendDiscussionTopicFunc(ch chan *DiscussionTopic) sendFunc {
	return func(r io.Reader) error {
		discs := make([]*DiscussionTopic, 0)
		if err := json.NewDecoder(r).Decode(&discs); err != nil {
			return err
		}
		for _, d := range discs {
			ch <- d
		}
		return nil
	}
}
