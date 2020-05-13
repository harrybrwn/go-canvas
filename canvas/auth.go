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

func (c *client) get(endpoint string, vals encoder) (*http.Response, error) {
	return get(c, endpoint, vals)
}

type doer interface {
	Do(*http.Request) (*http.Response, error)
}

func get(client doer, endpoint string, vals encoder) (*http.Response, error) {
	var q string
	if vals != nil {
		q = vals.Encode()
	}
	return client.Do(&http.Request{
		Method: "GET",
		Proto:  "HTTP/1.1",
		URL: &url.URL{
			Path:     path.Join("/api/v1", endpoint),
			RawQuery: q,
		},
	})
}

func put(client doer, endpoint string, vals encoder) (*http.Response, error) {
	var q string
	if vals != nil {
		q = vals.Encode()
	}
	return client.Do(&http.Request{
		Method: "PUT",
		Proto:  "HTTP/1.1",
		URL: &url.URL{
			Path:     path.Join("/api/v1", endpoint),
			RawQuery: q,
		},
	})
}

func getjson(client doer, obj interface{}, path string, vals encoder) error {
	resp, err := get(client, path, vals)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(obj)
}

func (c *client) getjson(obj interface{}, endpoint string, vals encoder) error {
	return getjson(c, obj, endpoint, vals)
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

func checkErrors(errs []*errorMsg) error {
	if len(errs) < 1 {
		return nil
	}
	msgs := make([]string, len(errs))
	for i := 0; i < len(errs); i++ {
		msgs[i] = fmt.Sprintf("canvas: %s", errs[i].Message)
	}
	return errors.New(strings.Join(msgs, ", "))
}

type authError struct {
	Status string
	Errors []*errorMsg
}

func (ae *authError) Error() string {
	return fmt.Sprintf("%s: %v", ae.Status, checkErrors(ae.Errors))
}

type errorMsg struct {
	Message string
}

func (em *errorMsg) Error() string {
	return em.Message
}
