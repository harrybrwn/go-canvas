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

type sendFunc func(io.Reader) error

func newPaginatedList(
	d doer,
	path string,
	send sendFunc,
	parameters []Option,
) *paginated {
	if parameters == nil {
		parameters = []Option{}
	}
	return &paginated{
		do:      d,
		path:    path,
		opts:    parameters,
		send:    send,
		perpage: 10,
		wg:      new(sync.WaitGroup),
		errs:    make(chan error),
	}
}

type paginated struct {
	path string
	opts []Option
	do   doer
	send sendFunc

	perpage int
	errs    chan error

	wg *sync.WaitGroup
}

type closable interface {
	Close()
}

type errorHandlerFunc func(error) error

// Possible bug: ok so this function should be run in a sperate goroutine. When an
// error is found and the send channel 'ch' is closed, some objects may be sent
// on the channel after it is closed because it was closed in a seperate goroutine.
func handleErrs(errs <-chan error, ch closable, handle errorHandlerFunc) {
	var err error
	for {
		select {
		case e := <-errs:
			// If e is nil, the error channel has been closed and we stop
			// otherwise we handle the error.
			if e != nil {
				// If the user defined error returns an error then we stop,
				// if it returns nil, then the user wants to keep going and
				// handle the error one their side.
				err = handle(e)
				if err != nil {
					goto Stop
				}
				continue // don't stop just for one error
			}
		Stop:
			ch.Close() // ch should be a chan wrapped in a type
			return
		}
	}
}

// returns <number of pages>, <first response>
func (p *paginated) firstReq() (int, *http.Response, error) {
	resp, err := get(p.do, p.path, p.getQuery(1))
	if err != nil {
		return 0, nil, err
	}
	n, err := findlastpage(resp.Header)
	if err != nil {
		return 0, nil, err
	}
	return n, resp, nil
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
		if err = p.send(resp.Body); err != nil {
			p.errs <- err
		}
		resp.Body.Close()
		p.wg.Done()
	}()
	// already made a request for page 1, so start on 2
	for page := 2; page <= n; page++ {
		go func(page int) {
			defer p.wg.Done()
			resp, err := get(p.do, p.path, p.getQuery(page))
			if err != nil {
				p.errs <- err
				return // stop bc we won't have data to send
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
	q.Add(p.opts)
	return q
}

var (
	resourceRegex = regexp.MustCompile(`<(.*?)>; rel="(.*?)"`)
	lastpageRegex = regexp.MustCompile(`.*<(.*)[\?&]page=([0-9]*).*>; rel="last"`)
)

func findlastpage(header http.Header) (int, error) {
	links := header.Get("Link")
	if links == "" {
		return -1, errs.New("this is not a request for a paginated list")
	}
	parts := lastpageRegex.FindStringSubmatch(links)
	if len(parts) < 3 {
		return -1, errs.New("could not find last page")
	}
	page, err := strconv.ParseInt(parts[2], 10, 32)
	return int(page), err
}

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
