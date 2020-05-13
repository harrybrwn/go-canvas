package canvas

import (
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
	settings := make(map[string]interface{})
	return settings, getjson(u.client, &settings, fmt.Sprintf("/users/%d/settings", u.ID), nil)
}

// UserColor is just a hex color.
type UserColor struct {
	HexCode string `json:"hexcode"`
}

// GetColors will return a map of the user's custom profile colors.
func (u *User) GetColors() (map[string]string, error) {
	colors := make(map[string]map[string]string)
	err := getjson(u.client, &colors, fmt.Sprintf("users/%d/colors", u.ID), nil)
	if err != nil {
		return nil, err
	}
	return colors["custom_colors"], nil
}

// GetColor will get a specific color from the user's profile.
func (u *User) GetColor(asset string) (UserColor, error) {
	color := UserColor{}
	return color, getjson(u.client, &color, fmt.Sprintf("users/%d/colors/%s", u.ID, asset), nil)
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
	return nil
}
