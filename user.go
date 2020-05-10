package canvas

import "time"

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
func (u *User) Settings() error {
	panic("not implimented")
}
