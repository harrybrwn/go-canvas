package canvas

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Course options are given when requesting courses in order to
// filter out certain courses that may not be wanted in the query.
var (
	CompletedCourses        Option = Opt("enrollment_state", "completed")
	ActiveCourses           Option = Opt("enrollment_state", "active")
	InvitedOrPendingCourses Option = Opt("enrollment_state", "invited_or_pending")
)

// Enrollment options are given to filter out different types of people
var (
	OptTeacher  Option = Opt("enrollment_type", "teacher")
	OptStudent  Option = Opt("enrollment_type", "student")
	OptTA       Option = Opt("enrollment_type", "ta")
	OptObserver Option = Opt("enrollment_type", "observer")
	OptDesigner Option = Opt("enrollment_type", "designer")
)

// Option is a key value pair used
// for api parameters. see Opt
type Option interface {
	Name() string
	Value() []string
}

// Opt creates a new option. Used for creating
// a new Param interface.
func Opt(key string, val interface{}) Option {
	return &option{key, val}
}

// ArrayOpt creates an option that will be sent as an
// array of options (ex. include[], content_type[], etc.)
func ArrayOpt(key string, vals ...string) Option {
	return &arropt{
		key:  fmt.Sprintf("%s[]", key),
		vals: vals,
	}
}

const dateFormat = time.RFC3339

// DateOpt will return an Option with a correctly formatted date.
func DateOpt(key string, date time.Time) Option {
	return &option{
		key: key,
		val: date.Format(dateFormat),
	}
}

// IncludeOpt is the option for any "include[]" api
// parameters.
func IncludeOpt(vals ...string) Option {
	return &arropt{
		key:  "include[]",
		vals: vals,
	}
}

// SortOpt returns a sorting option
func SortOpt(schemes ...string) Option {
	return ArrayOpt("sort", schemes...)
}

// ContentTypes retruns a option param for getting a content type.
// As a rule of thumb this is used for searches or filtering.
func ContentTypes(contentTypes ...string) Option {
	return ArrayOpt("content_types", contentTypes...)
}

// ContentType retruns a option param for getting a content type.
// Generally used for file uploads.
func ContentType(contentType string) Option {
	return Opt("content_type", contentType)
}

// UserOpt creates an Option that should be sent
// when asking for a user, updating a user, or creating a user.
func UserOpt(key, val string) Option {
	return &prefixedOption{key: key, val: []string{val}, prefix: "user"}
}

func asPrefixed(prefix string, opt Option) Option {
	return &prefixedOption{
		key:    opt.Name(),
		val:    opt.Value(),
		prefix: prefix,
	}
}

func toPrefixedOpts(prefix string, opts []Option) []Option {
	prefixed := make([]Option, len(opts))
	for i := range opts {
		prefixed[i] = asPrefixed(prefix, opts[i])
	}
	return prefixed
}

type prefixedOption struct {
	key, prefix string
	val         []string
}

func (po *prefixedOption) Name() string {
	return fmt.Sprintf("%s[%s]", po.prefix, po.key)
}

func (po *prefixedOption) Value() []string {
	return po.val
}

type option struct {
	key string
	val interface{}
}

func (o *option) Name() string {
	return o.key
}

func (o *option) Value() []string {
	return []string{fmt.Sprintf("%v", o.val)}
}

type arropt struct {
	key  string
	vals []string
}

func (ao *arropt) Name() string {
	return ao.key
}

func (ao *arropt) Value() []string {
	return ao.vals
}

// Encodable options
type optEnc []Option

func (oe optEnc) Encode() string {
	if len(oe) == 0 {
		return ""
	}
	var buf strings.Builder
	for _, o := range oe {
		if buf.Len() > 0 {
			buf.WriteByte('&')
		}
		buf.WriteString(url.QueryEscape(o.Name()))
		buf.WriteByte('=')
		buf.WriteString(url.QueryEscape(strings.Join(o.Value(), ",")))
	}
	return buf.String()
}
