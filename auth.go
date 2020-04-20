package canvas

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
)

// DefaultHost is the default url host for the canvas api.
var DefaultHost = "canvas.instructure.com"

func newclient(token string) *client {
	return &client{
		Client: http.Client{
			Transport: &auth{
				rt:    http.DefaultTransport,
				token: token,
				host:  DefaultHost,
			},
		},
	}
}

type client struct {
	http.Client
}

func (c *client) get(endopint string, vals encoder) (*http.Response, error) {
	var q string
	if vals != nil {
		q = vals.Encode()
	}

	return c.Do(&http.Request{
		Method: "GET",
		Proto:  "HTTP/1.1",
		URL: &url.URL{
			Path:     path.Join("/api/v1", endopint),
			RawQuery: q,
		},
	})
}

func (c *client) getjson(obj interface{}, endpoint string, vals encoder) error {
	resp, err := c.get(endpoint, vals)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(obj)
}

type hasclient interface {
	setClient(*client)
}

type auth struct {
	rt    http.RoundTripper
	token string
	host  string
}

func (a *auth) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.token))
	req.Host = a.host
	req.URL.Scheme = "https"
	req.URL.Host = a.host
	return a.rt.RoundTrip(req)
}

type encoder interface {
	Encode() string
}
