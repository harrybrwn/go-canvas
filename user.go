package canvas

import (
	"fmt"
	"io"
	"path"
	"path/filepath"
	"time"
)

// User is a canvas user
type User struct {
	ID              int        `json:"id"`
	Name            string     `json:"name"`
	Email           string     `json:"email"`
	Bio             string     `json:"bio"`
	SortableName    string     `json:"sortable_name"`
	ShortName       string     `json:"short_name"`
	SisUserID       string     `json:"sis_user_id"`
	SisImportID     int        `json:"sis_import_id"`
	IntegrationID   string     `json:"integration_id"`
	CreatedAt       time.Time  `json:"created_at"`
	LoginID         string     `json:"login_id"`
	AvatarURL       string     `json:"avatar_url"`
	Enrollments     Enrollment `json:"enrollments"`
	Locale          string     `json:"locale"`
	EffectiveLocale string     `json:"effective_locale"`
	LastLogin       time.Time  `json:"last_login"`
	TimeZone        string     `json:"time_zone"`

	CanUpdateAvatar bool `json:"can_update_avatar"`
	Permissions     struct {
		CanUpdateName           bool `json:"can_update_name"`
		CanUpdateAvatar         bool `json:"can_update_avatar"`
		LimitParentAppWebAccess bool `json:"limit_parent_app_web_access"`
	} `json:"permissions"`
	client doer
}

// Settings will get the user's settings.
func (u *User) Settings() (settings map[string]interface{}, err error) {
	// TODO: find the settings json response and use a struct not a map
	return settings, getjson(u.client, &settings, nil, "/users/%d/settings", u.ID)
}

// Courses will return the user's courses.
func (u *User) Courses(opts ...Option) ([]*Course, error) {
	return getCourses(u.client, u.id("/users/%d/courses"), optEnc(opts))
}

// FavoriteCourses returns the user's list of favorites courses.
func (u *User) FavoriteCourses(opts ...Option) ([]*Course, error) {
	return getCourses(u.client, "/users/favorites/courses", optEnc(opts))
}

// File will get a user's file by id
func (u *User) File(id int, opts ...Option) (*File, error) {
	return getUserFile(u.client, id, u.ID, opts)
}

// Files will return a channel of files.
func (u *User) Files(opts ...Option) <-chan *File {
	return filesChannel(
		u.client, u.id("/users/%d/files"),
		ConcurrentErrorHandler, opts, nil,
	)
}

// ListFiles will collect all of the users files.
func (u *User) ListFiles(opts ...Option) ([]*File, error) {
	return listFiles(u.client, u.id("/users/%d/files"), nil, opts)
}

// Folders returns a channel of the user's folders.
func (u *User) Folders(opts ...Option) <-chan *Folder {
	return foldersChannel(
		u.client, u.id("/users/%d/folders"),
		ConcurrentErrorHandler, opts, nil,
	)
}

// Root will get the root folder for the user's files.
func (u *User) Root(opts ...Option) (*Folder, error) {
	f := &Folder{client: u.client}
	return f, getjson(u.client, f, optEnc(opts), "/users/%d/folders/root", u.ID)
}

// ListFolders will return a slice of all the user's folders
func (u *User) ListFolders(opts ...Option) ([]*Folder, error) {
	return listFolders(u.client, u.id("/users/%d/folders"), nil, opts)
}

// FolderPath will split the path and return a list containing
// all of the folders in the path.
func (u *User) FolderPath(pth string) ([]*Folder, error) {
	pth = path.Join(u.id("/users/%d/folders/by_path"), pth)
	return folderList(u.client, pth)
}

// UploadFile will upload the contents of an io.Reader to a
// new file in the user's files.
func (u *User) UploadFile(
	filename string,
	r io.Reader,
	opts ...Option,
) (*File, error) {
	return uploadFile(u.client, filename, r, u.id("/users/%d/files"), opts)
}

// CreateFolder will create a new folder.
func (u *User) CreateFolder(path string, opts ...Option) (*Folder, error) {
	dir, name := filepath.Split(path)
	return createFolder(
		u.client,
		dir,
		name,
		opts,
		"/users/%d/folders", u.ID,
	)
}

// ContextCode returns the context code for the user.
func (u *User) ContextCode() string {
	return fmt.Sprintf("user_%d", u.ID)
}

// CalendarEvents gets the user's calendar events.
func (u *User) CalendarEvents(opts ...Option) (cal []CalendarEvent, err error) {
	return cal, getjson(u.client, &cal, optEnc(opts), "/users/%d/calendar_events", u.ID)
}

// Bookmarks will get the user's bookmarks
func (u *User) Bookmarks(opts ...Option) (bks []Bookmark, err error) {
	return bks, getjson(u.client, &bks, optEnc(opts), "users/%d/bookmarks", u.ID)
}

// CreateBookmark will create a bookmark
func (u *User) CreateBookmark(b *Bookmark) error {
	return createBookmark(u.client, u.ID, b)
}

// DeleteBookmark will delete a user's bookmark.
func (u *User) DeleteBookmark(b *Bookmark) error {
	return deleteBookmark(u.client, u.ID, b.ID)
}

// Profile will make a call to get the user's profile data.
func (u *User) Profile() (p *UserProfile, err error) {
	return p, getjson(u.client, p, nil, "/users/%d/profile", u.ID)
}

// UserProfile is a user's profile data.
type UserProfile struct {
	ID             int               `json:"id"`
	LoginID        string            `json:"login_id"`
	Name           string            `json:"name"`
	PrimaryEmail   string            `json:"primary_email"`
	ShortName      string            `json:"short_name"`
	SortableName   string            `json:"sortable_name"`
	TimeZone       string            `json:"time_zone"`
	Bio            string            `json:"bio"`
	Title          string            `json:"title"`
	Calendar       map[string]string `json:"calendar"`
	LTIUserID      string            `json:"lti_user_id"`
	AvatarURL      string            `json:"avatar_url"`
	EffectiveLocal string            `json:"effective_local"`
	IntegrationID  string            `json:"integration_id"`
	Local          string            `json:"local"`
}

// GradedSubmissions gets the user's graded submissions.
func (u *User) GradedSubmissions() (subs []*Submission, err error) {
	return subs, getjson(u.client, &subs, nil, "/users/%d/graded_submissions", u.ID)
}

// Submission is a submission type.
type Submission struct {
	AssignmentID                  int         `json:"assignment_id"`
	Assignment                    interface{} `json:"assignment"`
	Course                        interface{} `json:"course"`
	Attempt                       int         `json:"attempt"`
	Body                          string      `json:"body"`
	Grade                         string      `json:"grade"`
	GradeMatchesCurrentSubmission bool        `json:"grade_matches_current_submission"`
	HTMLURL                       string      `json:"html_url"`
	PreviewURL                    string      `json:"preview_url"`
	Score                         float64     `json:"score"`
	SubmissionComments            interface{} `json:"submission_comments"`
	SubmissionType                string      `json:"submission_type"`
	SubmittedAt                   time.Time   `json:"submitted_at"`
	URL                           interface{} `json:"url"`
	UserID                        int         `json:"user_id"`
	GraderID                      int         `json:"grader_id"`
	GradedAt                      time.Time   `json:"graded_at"`
	User                          interface{} `json:"user"`
	Late                          bool        `json:"late"`
	AssignmentVisible             bool        `json:"assignment_visible"`
	Excused                       bool        `json:"excused"`
	Missing                       bool        `json:"missing"`
	LatePolicyStatus              string      `json:"late_policy_status"`
	PointsDeducted                float64     `json:"points_deducted"`
	SecondsLate                   int         `json:"seconds_late"`
	WorkflowState                 string      `json:"workflow_state"`
	ExtraAttempts                 int         `json:"extra_attempts"`
	AnonymousID                   string      `json:"anonymous_id"`
	PostedAt                      time.Time   `json:"posted_at"`
}

// Avatars will get a list of the user's avatars.
func (u *User) Avatars() (av []Avatar, err error) {
	return av, getjson(u.client, &av, nil, "/users/%d/avatars", u.ID)
}

// Avatar is the avatar data for a user.
type Avatar struct {
	ID          int    `json:"id"`
	Type        string `json:"type"`
	DisplayName string `json:"display_name"`
	Filename    string `json:"filename"`
	URL         string `json:"url"`
	Token       string `json:"token"`
	ContentType string `json:"content-type"`
	Size        int    `json:"size"`
}

// UserColor is just a hex color.
type UserColor struct {
	HexCode string `json:"hexcode"`
}

// Colors will return a map of the user's custom profile colors.
func (u *User) Colors() (map[string]string, error) {
	colors := make(map[string]map[string]string)
	err := getjson(u.client, &colors, nil, "users/%d/colors", u.ID)
	if err != nil {
		return nil, err
	}
	return colors["custom_colors"], nil
}

// Color will get a specific color from the user's profile.
func (u *User) Color(asset string) (color *UserColor, err error) {
	return color, getjson(u.client, color, nil, "users/%d/colors/%s", u.ID, asset)
}

// SetColor will update the color of the given asset to as specific hex color.
func (u *User) SetColor(asset, hexcode string) error {
	path := fmt.Sprintf("users/%d/colors/%s", u.ID, asset)
	if hexcode[0] == '#' {
		hexcode = hexcode[1:]
	}

	resp, err := put(u.client, path, params{"hexcode": {hexcode}})
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func getUserFile(d doer, id int, userid interface{}, opts optEnc) (*File, error) {
	f := &File{client: d}
	return f, getjson(d, f, opts, "/users/%v/files/%d", userid, id)
}

func (u *User) id(s string) string {
	return fmt.Sprintf(s, u.ID)
}
