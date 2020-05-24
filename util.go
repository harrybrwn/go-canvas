package canvas

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func asParams(opts []Option) params {
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

func (p params) Add(vals ...Option) {
	for _, v := range vals {
		p[v.Name()] = v.Value()
	}
}

// Encode converts the params to a string
// representation of a url parameter.
func (p params) Encode() string {
	return url.Values(p).Encode()
}

// Params represents parameters passed to a url
type genericParam map[string]interface{}

// Encode converts the map alias to a string representation of a url parameter.
func (p genericParam) Encode() string {
	// I totally stole this function from the net/url package. I should probably
	// give credit where it is due.
	if p == nil {
		return ""
	}
	var value string
	var buffer strings.Builder
	for k, val := range p {
		key := url.QueryEscape(k)
		switch v := val.(type) {
		case int:
			value = strconv.Itoa(v)
		case string:
			value = v
		case []byte:
			value = string(v)
		case byte:
			value = string(v)
		case bool:
			value = strconv.FormatBool(v)
		case *time.Time:
			value = v.Format(dateFormat)
		default:
			panic(fmt.Sprintf("can't encode type %T", v))
		}

		if buffer.Len() > 0 {
			buffer.WriteByte('&')
		}
		buffer.WriteString(key)
		buffer.WriteByte('=')
		buffer.WriteString(url.QueryEscape(value))
	}
	return buffer.String()
}

func (p genericParam) addOpts(opts []Option) {
	for _, o := range opts {
		p[o.Name()] = o.Value()[0]
	}
}

var (
	_ encoder = (*params)(nil)
	_ encoder = (*genericParam)(nil)
)
