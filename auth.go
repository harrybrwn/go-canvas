package canvas

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/harrybrwn/errs"
)

var (
	// ErrRateLimitExceeded is returned when the api rate limit has been reached.
	ErrRateLimitExceeded = errors.New("403 Forbidden (Rate Limit Exceeded)")

	apiPath = "/api/v1"
)

// IsRateLimit returns true if the error
// given is a rate limit error.
func IsRateLimit(e error) bool {
	if e == ErrRateLimitExceeded {
		return true
	}
	return false
}

type client struct {
	http.Client
	host string
}

func (c *client) Do(r *http.Request) (*http.Response, error) {
	if r.URL.Host == "" {
		r.Host = c.host
		r.URL.Host = c.host
	}
	return c.Client.Do(r)
}

type doer interface {
	Do(*http.Request) (*http.Response, error)
}

func do(d doer, req *http.Request) (*http.Response, error) {
	resp, err := d.Do(req)
	if err != nil {
		return nil, err
	}

	var e error
	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted:
		return resp, err
	case http.StatusForbidden:
		resp.Body.Close()
		return nil, ErrRateLimitExceeded
	case http.StatusUnprocessableEntity:
		return nil, errs.Pair(resp.Body.Close(), errs.New(resp.Status))
	case http.StatusNotFound, http.StatusUnauthorized:
		e = &AuthError{}
	case http.StatusBadRequest, http.StatusInternalServerError:
		e = &Error{Status: resp.Status}
	}
	return nil, errs.Chain(e, json.NewDecoder(resp.Body).Decode(&e), resp.Body.Close())
}

func get(c doer, endpoint string, vals encoder) (*http.Response, error) {
	var q string
	if vals != nil {
		q = vals.Encode()
	}
	return do(c, newreq("GET", endpoint, q))
}

func put(c doer, endpoint string, vals encoder) (*http.Response, error) {
	var q string
	if vals != nil {
		q = vals.Encode()
	}
	return do(c, newreq("PUT", endpoint, q))
}

func post(c doer, endpoint string, vals encoder) (*http.Response, error) {
	var q string
	if vals != nil {
		q = vals.Encode()
	}
	return do(c, newreq("POST", endpoint, q))
}

func delete(c doer, endpoint string, vals encoder) (*http.Response, error) {
	var q string
	if vals != nil {
		q = vals.Encode()
	}
	return do(c, newreq("DELETE", endpoint, q))
}

func newreq(method, urlpath, query string) *http.Request {
	return newV1Req(method, urlpath, query)
}

func newV1Req(method, urlpath, query string) *http.Request {
	return &http.Request{
		Method: method,
		Proto:  "HTTP/1.1",
		URL: &url.URL{
			Scheme:   "https",
			Path:     path.Join(apiPath, urlpath),
			RawQuery: query,
		},
	}
}

func getjson(
	client doer,
	obj interface{},
	vals encoder,
	path string,
	v ...interface{},
) error {
	resp, err := get(client, fmt.Sprintf(path, v...), vals)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(obj)
}

func authorize(c *http.Client, token, host string) {
	rt := http.DefaultTransport
	if c.Transport != nil {
		rt = c.Transport
	}
	c.Transport = &auth{
		rt:    rt,
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
	req.Header.Set("User-Agent", DefaultUserAgent)
	if req.URL.Host == "" {
		// TODO: don't do this, it has caused my too much pain
		req.Host = a.host
		req.URL.Host = a.host
	}
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

// Error is an error response.
type Error struct {
	Errors struct {
		EndDate string `json:"end_date"`
	} `json:"errors"`
	Message string `json:"message"`

	Err      string `json:"error"`
	SentryID string `json:"sentryId"`

	Status string `json:"-"`
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Errors.EndDate != "" {
		return fmt.Sprintf("end_date: %s", e.Errors.EndDate)
	}
	if e.SentryID != "" {
		return fmt.Sprintf("error status: %s; sentryId: %s", e.Err, e.SentryID)
	}
	return fmt.Sprintf("canvas error: %#v", e)
}

// AuthError is an authentication error response from canvas.
type AuthError struct {
	Status string     `json:"status"`
	Errors []errorMsg `json:"errors"`
}

func (ae *AuthError) Error() string {
	if ae.Status == "" {
		return checkErrors(ae.Errors)
	}
	return fmt.Sprintf("%s: %s", ae.Status, checkErrors(ae.Errors))
}

type errorMsg struct {
	Message string `json:"message,omitempty"`
}
