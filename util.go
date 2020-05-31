package canvas

import (
	"net/url"
)

type params map[string][]string

type encoder interface {
	Encode() string
}

func (p params) Add(vals []Option) {
	for _, v := range vals {
		p[v.Name()] = v.Value()
	}
}

func (p params) Set(key, val string) {
	p[key] = []string{val}
}

// Encode converts the params to a string
// representation of a url parameter.
func (p params) Encode() string {
	return url.Values(p).Encode()
}

func pathFromContextType(contextType string) string {
	switch contextType {
	case "Course":
		return "courses"
	case "User":
		return "users"
	case "GroupCategory":
		return "group_categories"
	case "Account":
		return "accounts"
	default:
		return ""
	}
}

var _ encoder = (*params)(nil)
