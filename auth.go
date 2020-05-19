package canvas

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// ErrRateLimitExceeded is returned when the api rate limit has been reached.
var ErrRateLimitExceeded = errors.New("403 Forbidden (Rate Limit Exceeded)")

// IsRateLimit returns true if the error
// given is a rate limit error.
func IsRateLimit(e error) bool {
	if e == nil {
		return false
	}
	if e == ErrRateLimitExceeded {
		return true
	}
	return strings.Contains(e.Error(), "Rate Limit Exceeded")
}

type client struct {
	http.Client
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
	case http.StatusOK:
		return resp, err
	case http.StatusNotFound, http.StatusUnauthorized:
		e = &AuthError{}
	case http.StatusBadRequest:
		e = &Error{}
	case http.StatusForbidden:
		resp.Body.Close()
		rem := resp.Header.Get("X-Rate-Limit-Remaining")
		return nil, fmt.Errorf(
			"%s; remaining: %s", ErrRateLimitExceeded.Error(), rem)
	}
	err = errpair(e, json.NewDecoder(resp.Body).Decode(&e))
	return nil, errpair(err, resp.Body.Close())
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

func getjson(client doer, obj interface{}, vals encoder, path string, v ...interface{}) error {
	resp, err := get(client, fmt.Sprintf(path, v...), vals)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(obj)
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
	req.Header.Set("User-Agent", DefaultUserAgent)
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

// Error is an error response.
type Error struct {
	Errors struct {
		EndDate string `json:"end_date"`
	} `json:"errors"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Errors.EndDate != "" {
		return fmt.Sprintf("end_date: %s", e.Errors.EndDate)
	}
	return "canvas error"
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
