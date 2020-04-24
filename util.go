package canvas

import (
	"fmt"
	"net/url"
	"strings"
)

// Param is a url parameter.
type Param interface {
	Name() string
	Value() []string
}

// Opt creates a new option. Used for creating
// a new Param interface.
func Opt(key, val string) Param {
	return &option{key, val}
}

// ArrayOpt creates an option that will be sent as an
// array of options (ex. include[], content_type[], etc.)
func ArrayOpt(key string, vals ...string) Param {
	return &option{
		key: fmt.Sprintf("%s[]", key),
		val: strings.Join(vals, ","),
	}
}

// SortOpt returns a sorting option
func SortOpt(schemes ...string) Param {
	return ArrayOpt("sort", schemes...)
}

// ContentType retruns a option param for getting a content type.
func ContentType(contentType string) Param {
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

func makeparams(opts ...Param) params {
	p := params{}
	for _, o := range opts {
		p[o.Name()] = o.Value()
	}
	return p
}

type params map[string][]string

type encoder interface {
	Encode() string
}

func (p params) Join(pa map[string][]string) {
	for k, v := range pa {
		if _, ok := p[k]; ok {
			continue
		}
		p[k] = v
	}
}

func (p params) Add(vals ...Param) {
	for _, v := range vals {
		p[v.Name()] = v.Value()
	}
}

// Encode converts the params to a string
// representation of a url parameter.
func (p params) Encode() string {
	return url.Values(p).Encode()
}

var _ encoder = (*params)(nil)
