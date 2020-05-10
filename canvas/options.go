package canvas

import (
	"fmt"
	"strings"
)

// Option is a key value pair used
// for api parameters. see Opt
type Option interface {
	Name() string
	Value() []string
}

// Opt creates a new option. Used for creating
// a new Param interface.
func Opt(key, val string) Option {
	return &option{key, val}
}

// ArrayOpt creates an option that will be sent as an
// array of options (ex. include[], content_type[], etc.)
func ArrayOpt(key string, vals ...string) Option {
	return &option{
		key: fmt.Sprintf("%s[]", key),
		val: strings.Join(vals, ","),
	}
}

// SortOpt returns a sorting option
func SortOpt(schemes ...string) Option {
	return ArrayOpt("sort", schemes...)
}

// ContentType retruns a option param for getting a content type.
func ContentType(contentType string) Option {
	return ArrayOpt("content_types", contentType)
}

type option struct {
	key, val string
}

func (o *option) Name() string {
	return o.key
}

func (o *option) Value() []string {
	return []string{o.val}
}
