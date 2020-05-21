package canvas

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"sync"

	"github.com/harrybrwn/errs"
)

type pageInitFunction func(int, io.Reader) ([]interface{}, error)

type sendFunc func(io.Reader) error

func newPaginatedList(
	d doer,
	path string,
	send func(io.Reader) error,
	parameters []Option,
) *paginated {
	if parameters == nil {
		parameters = []Option{}
	}
	return &paginated{
		do:      d,
		path:    path,
		query:   asParams(parameters),
		send:    send,
		perpage: 10,
		wg:      new(sync.WaitGroup),
		errs:    make(chan error),
	}
}

type paginated struct {
	path  string
	query params
	do    doer
	send  func(io.Reader) error

	n       int
	perpage int
	errs    chan error

	wg *sync.WaitGroup
}

// returns <number of pages>, <first response>
func (p *paginated) firstReq() (int, *http.Response, error) {
	q := params{"page": {"1"}, "per_page": {fmt.Sprintf("%d", p.perpage)}}
	q.Join(p.query)
	resp, err := get(p.do, p.path, q)
	if err != nil {
		return 0, nil, err
	}
	pages, err := newLinkedResource(resp.Header)
	if err != nil {
		return 0, nil, err
	}
	p.n = pages.Last.page
	return p.n, resp, nil
}

func (p *paginated) start() <-chan error {
	n, resp, err := p.firstReq() // n pages and first request
	if err != nil {
		go func() {
			p.errs <- err
			close(p.errs)
		}()
		return p.errs
	}
	p.wg.Add(n)

	go func() {
		defer resp.Body.Close()
		defer p.wg.Done()
		if err = p.send(resp.Body); err != nil {
			p.errs <- err
		}
	}()
	for page := 2; page <= n; page++ {
		go func(page int) {
			defer p.wg.Done()
			resp, err := get(p.do, p.path, p.getQuery(page))
			if err != nil {
				p.errs <- err
				return
			}
			if err = p.send(resp.Body); err != nil {
				p.errs <- err
			}
			resp.Body.Close()
		}(page)
	}
	go func() {
		p.wg.Wait()
		close(p.errs)
	}()
	return p.errs
}

func (p *paginated) getQuery(page int) params {
	q := params{
		"page":     {strconv.FormatInt(int64(page), 10)}, // base 10
		"per_page": {fmt.Sprintf("%d", p.perpage)},
	}
	q.Join(p.query)
	return q
}

// PagedError is an error type returned by paginated lists
// and contains an error channel.
type PagedError struct {
	Errs  chan error
	Count int
}

func (pe *PagedError) send(e error) {
	if e != nil {
		pe.Errs <- e
		pe.Count++
	}
}

// Errors returns a channel of errors.
func (pe *PagedError) Errors() <-chan error {
	return pe.Errs
}

func (pe *PagedError) Error() string {
	return fmt.Sprintf("found %d pagination errors in a %T", pe.Count, *pe)
}

func (p *paginated) ordered() ([]interface{}, error) {
	return nil, nil
}

var resourceRegex = regexp.MustCompile(`<(.*?)>; rel="(.*?)"`)

func newLinkedResource(header http.Header) (*linkedResource, error) {
	var err error
	res := &linkedResource{}
	links := header.Get("Link")
	parts := resourceRegex.FindAllStringSubmatch(links, -1)
	m := map[string]*link{}

	for _, part := range parts {
		m[part[2]], err = newlink(part[1])
		if err != nil {
			return res, err
		}
	}
	var ok bool
	if res.Current, ok = m["current"]; !ok {
		return nil, errs.New("could not find current link")
	}
	if res.First, ok = m["first"]; !ok {
		return nil, errs.New("could not find first link")
	}
	if res.Last, ok = m["last"]; !ok {
		return nil, errs.New("could not find last link")
	}
	res.Next, _ = m["next"]
	return res, nil
}

type linkedResource struct { //map[string]*link
	Current, First, Last, Next *link
}

type link struct {
	url  *url.URL
	page int
}

func newlink(urlstr string) (*link, error) {
	u, err := url.Parse(urlstr)
	if err != nil {
		return nil, err
	}
	page, err := strconv.ParseInt(u.Query().Get("page"), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("could not parse page num: %w", err)
	}
	return &link{
		url:  u,
		page: int(page),
	}, nil
}
