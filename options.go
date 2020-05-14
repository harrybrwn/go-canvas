package canvas

import (
	"fmt"
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
	return &arropt{
		key:  fmt.Sprintf("%s[]", key),
		vals: vals,
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

// ContentType retruns a option param for getting a content type.
func ContentType(contentTypes ...string) Option {
	return ArrayOpt("content_types", contentTypes...)
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
	key, val string
}

func (o *option) Name() string {
	return o.key
}

func (o *option) Value() []string {
	return []string{o.val}
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
