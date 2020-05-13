package canvas

import (
	"net/url"
)

func makeparams(opts ...Option) params {
	return asParams(opts)
}

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

var _ encoder = (*params)(nil)
