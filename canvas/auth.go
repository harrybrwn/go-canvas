package canvas

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
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

type doer interface {
	Do(*http.Request) (*http.Response, error)
}

func get(c doer, endpoint string, vals encoder) (*http.Response, error) {
	var q string
	if vals != nil {
		q = vals.Encode()
	}
	return c.Do(newreq("GET", endpoint, q))
}

func put(c doer, endpoint string, vals encoder) (*http.Response, error) {
	var q string
	if vals != nil {
		q = vals.Encode()
	}
	return c.Do(newreq("PUT", endpoint, q))
}

func post(c doer, endpoint string, vals encoder) (*http.Response, error) {
	var q string
	if vals != nil {
		q = vals.Encode()
	}
	return c.Do(newreq("POST", endpoint, q))
}

func newreq(method, urlpath, query string) *http.Request {
	return &http.Request{
		Method: method,
		Proto:  "HTTP/1.1",
		URL: &url.URL{
			Path:     path.Join("/api/v1", urlpath),
			RawQuery: query,
		},
	}
}

func getjson(client doer, obj interface{}, path string, vals encoder) error {
	resp, err := get(client, path, vals)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(obj)
}

type hasclient interface {
	setClient(*client)
}

func authorize(c *http.Client, token, host string) {
	c.Transport = &auth{
		rt:    http.DefaultTransport,
		token: token,
		host:  host,
	}
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

func checkErrors(errs []errorMsg) string {
	if len(errs) < 1 {
		return ""
	}
	msgs := make([]string, len(errs))
	for i := 0; i < len(errs); i++ {
		msgs[i] = fmt.Sprintf("%s", errs[i].Message)
	}
	return strings.Join(msgs, ", ")
}

// AuthError is an authentication error response from canvas.
type AuthError struct {
	Status string
	Errors []errorMsg
}

func (ae *AuthError) Error() string {
	if ae.Status == "" {
		return checkErrors(ae.Errors)
	}
	return fmt.Sprintf("%s: %s", ae.Status, checkErrors(ae.Errors))
}

type errorMsg struct {
	Message string
}
