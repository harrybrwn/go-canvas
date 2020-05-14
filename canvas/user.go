package canvas

import (
	"encoding/json"
	"fmt"
	"time"
)

// User is a user
type User struct {
	ID              int         `json:"id"`
	Name            string      `json:"name"`
	SortableName    string      `json:"sortable_name"`
	ShortName       string      `json:"short_name"`
	SisUserID       string      `json:"sis_user_id"`
	SisImportID     int         `json:"sis_import_id"`
	CreatedAt       time.Time   `json:"created_at"`
	IntegrationID   string      `json:"integration_id"`
	LoginID         string      `json:"login_id"`
	AvatarURL       string      `json:"avatar_url"`
	Enrollments     interface{} `json:"enrollments"`
	Email           string      `json:"email"`
	Locale          string      `json:"locale"`
	EffectiveLocale string      `json:"effective_locale"`
	LastLogin       time.Time   `json:"last_login"`
	TimeZone        string      `json:"time_zone"`
	Bio             string      `json:"bio"`

	CanUpdateAvatar bool `json:"can_update_avatar"`
	Permissions     struct {
		CanUpdateName           bool `json:"can_update_name"`
		CanUpdateAvatar         bool `json:"can_update_avatar"`
		LimitParentAppWebAccess bool `json:"limit_parent_app_web_access"`
	} `json:"permissions"`

	client doer
}

// Settings will get the user's settings.
func (u *User) Settings() (map[string]interface{}, error) {
	// TODO: find the settings json response and use a struct not a map
	settings := make(map[string]interface{})
	if err := getjson(u.client, &settings, fmt.Sprintf("/users/%d/settings", u.ID), nil); err != nil {
		return nil, err
	}
	errors, eok := settings["errors"]
	status, sok := settings["status"]
	if eok || sok {
		return nil, fmt.Errorf("%s: %v", status, errors)
	}
	return settings, nil
}

// Bookmarks will get the user's bookmarks
func (u *User) Bookmarks(opts ...Option) ([]Bookmark, error) {
	return getBookmarks(u.client, u.ID, opts)
}

// Profile will make a call to get the user's profile data.
func (u *User) Profile() (*UserProfile, error) {
	res := struct {
		*UserProfile
		*AuthError
	}{nil, nil}
	if err := getjson(u.client, &res, fmt.Sprintf("/users/%d/profile", u.ID), nil); err != nil {
		return nil, err
	}
	if res.AuthError != nil {
		return nil, res.AuthError
	}
	return res.UserProfile, nil
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
func (u *User) GradedSubmissions() ([]*Submission, error) {
	var submissions []*Submission
	return submissions, getjson(u.client, &submissions, fmt.Sprintf("/users/%d/graded_submissions", u.ID), nil)
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
func (u *User) Avatars() ([]Avatar, error) {
	avatars := []Avatar{}
	if err := getjson(u.client, &avatars, fmt.Sprintf("/users/%d/avatars", u.ID), nil); err != nil {
		return nil, err
	}
	return avatars, nil
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
	err := getjson(u.client, &colors, fmt.Sprintf("users/%d/colors", u.ID), nil)
	if err != nil {
		return nil, err
	}
	return colors["custom_colors"], nil
}

// Color will get a specific color from the user's profile.
func (u *User) Color(asset string) (*UserColor, error) {
	res := struct {
		*UserColor
		*AuthError
	}{nil, nil}
	err := getjson(u.client, &res, fmt.Sprintf("users/%d/colors/%s", u.ID, asset), nil)
	if err != nil {
		return nil, nil
	}
	if res.AuthError != nil {
		return nil, res.AuthError
	}
	return res.UserColor, nil
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
	defer resp.Body.Close()
	e := &AuthError{}
	return errpair(json.NewDecoder(resp.Body).Decode(e), e)
}
