package canvas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/harrybrwn/errs"
)

func TestPager(t *testing.T) {
	client := &http.Client{}
	authorize(client, testToken(), DefaultHost)

	req := newreq("GET", "/users/self/files", "")
	resp, err := client.Do(req)
	if err != nil {
		t.Error("could not do request:", err)
	}
	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	resp.Body.Close()
	linked, err := newLinkedResource(resp.Header)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(linked.Last.url)

	files := make([]*File, 0, 10)
	json.NewDecoder(&buf).Decode(&files)
	for _, f := range files {
		fmt.Println(f.ID, f.Filename)
	}
}

func newSyncAuth(a *auth, mu *sync.Mutex) *syncAuth {
	return &syncAuth{
		tok: a.token,
		rt:  http.DefaultTransport,
		mu:  mu,
	}
}

type syncAuth struct {
	tok string
	rt  http.RoundTripper
	mu  *sync.Mutex
}

func (sa *syncAuth) RoundTrip(r *http.Request) (*http.Response, error) {
	sa.mu.Lock()
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sa.tok))
	r.Header.Set("User-Agent", "tests: "+DefaultUserAgent)
	r.Host = DefaultHost
	r.URL.Host = DefaultHost
	r.URL.Scheme = "https"
	resp, err := sa.rt.RoundTrip(r)
	sa.mu.Unlock()
	return resp, err
}

// i'm so sorry, but this mess is actually sort of usful for testing
func TestPaginationErrors(t *testing.T) {
	c := testCourse()
	tr := c.client.(*http.Client).Transport
	var mu sync.Mutex
	// we need a syncAuth for this test because some goroutines will be
	// modifying the client object
	if a, ok := tr.(*auth); ok {
		c.client.(*http.Client).Transport = newSyncAuth(a, &mu)
	}
	allfiles, err := c.ListFiles()
	if err != nil {
		t.Fatal(err)
	}
	var testerror = errs.New("test error")

	t.Run("send_error", func(t *testing.T) {
		readCount := 0
		ch := make(fileChan)
		send := func(r io.Reader) error {
			mu.Lock()
			readCount++
			if readCount == 4 {
				mu.Unlock()
				return testerror // send an error only after the first request
			}
			mu.Unlock()
			files := make([]*File, 0)
			err := json.NewDecoder(r).Decode(&files)
			for _, f := range files {
				ch <- f
			}
			return err
		}
		p := newPaginatedList(
			c.client, fmt.Sprintf("courses/%d/files/", c.ID),
			send, nil,
		)
		p.perpage = 4
		go handleErrs(p.start(), ch, func(e error) error {
			if e != testerror {
				t.Error("should only be handling the error I sent")
			}
			return nil
		})
		fileCount := 0
		for range ch {
			fileCount++
		}
		if readCount != 5 {
			t.Error("should have gone through all the pages")
		}
		if fileCount >= len(allfiles) {
			t.Error("should not have gotten all of the files")
		}
	})
	t.Run("auth_error", func(t *testing.T) {
		var tok string
		readCount := 0
		ch := make(fileChan)
		send := func(r io.Reader) error {
			mu.Lock()
			readCount++
			if readCount == 2 {
				tok = c.client.(*http.Client).Transport.(*syncAuth).tok
				c.client.(*http.Client).Transport.(*syncAuth).tok = ""
			}
			mu.Unlock()
			files := make([]*File, 0)
			err := json.NewDecoder(r).Decode(&files)
			for _, f := range files {
				ch <- f
			}
			return err
		}
		p := newPaginatedList(
			c.client,
			fmt.Sprintf("courses/%d/files/", c.ID),
			send, nil,
		)
		p.perpage = 4
		go handleErrs(p.start(), ch, func(e error) error {
			if e == nil {
				t.Error("expected error")
			}
			err, ok := e.(*AuthError)
			if !ok {
				t.Errorf("expected an auth error; got %T", err)
			}
			return nil
		})
		count := 0
		for f := range ch {
			if f.ID == 0 {
				t.Error("got bad file id")
			}
			count++
		}
		c.client.(*http.Client).Transport.(*syncAuth).tok = tok
		if count >= len(allfiles) {
			t.Error("should not have the same count")
		}
	})
}
