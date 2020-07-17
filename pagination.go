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

const (
	defaultPerPage = 10
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
		perpage: defaultPerPage,
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

// Possible bug: ok so this function should be run in a sperate goroutine.
// When an error is found and the send channel 'ch' is closed, some
// objects may be sent on the channel after it is closed because it was
// closed in a seperate goroutine.
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

type pageReader interface {
	io.Reader
	Page() int
}

type pagereader struct {
	num  int
	body io.Reader
}

func (p *pagereader) Page() int {
	return p.num
}

func (p *pagereader) Read(b []byte) (int, error) {
	return p.body.Read(b)
}

// returns <number of pages>, <first response>
func (p *paginated) firstReq() (int, *http.Response, error) {
	resp, err := get(p.do, p.path, p.getPageQuery(1))
	if err != nil {
		return -1, nil, err
	}
	n, err := findlastpage(resp.Header)
	if err != nil {
		return -1, nil, err
	}
	return n, resp, nil
}

func (p *paginated) start() <-chan error {
	n, resp, err := p.firstReq() // n pages and first request
	if err != nil || n == -1 {
		go func() {
			p.errs <- err
			p.Close()
		}()
		return p.errs
	}
	p.wg.Add(n)

	go func() {
		if err = p.send(&pagereader{0, resp.Body}); err != nil {
			p.errs <- err
		}
		resp.Body.Close()
		p.wg.Done()
	}()
	// Already made a request for page 1, so start on 2
	for page := 2; page <= n; page++ {
		go func(page int) {
			defer p.wg.Done()
			resp, err := get(p.do, p.path, p.getPageQuery(page))
			if err != nil {
				p.errs <- err
				return // stop bc we won't have data to send
			}
			// Using page - 1 because pagereaders index from 0 not 1
			if err = p.send(&pagereader{page - 1, resp.Body}); err != nil {
				p.errs <- err
			}
			resp.Body.Close()
		}(page)
	}
	go func() {
		p.wg.Wait()
		p.Close()
	}()
	return p.errs
}

func (p *paginated) Close() {
	close(p.errs)
}

func (p *paginated) getPageQuery(page int) params {
	q := params{
		"page":     {strconv.Itoa(page)},
		"per_page": {strconv.Itoa(p.perpage)},
	}
	q.Add(p.opts)
	return q
}

func getList(d doer, init func(io.Reader) error, path string, opts []Option) error {
	if opts == nil {
		opts = []Option{}
	}
	var (
		page    = 1
		perpage = 10
	)
	p := params{
		"page":     {strconv.Itoa(page)},
		"per_page": {strconv.Itoa(perpage)},
	}
	p.Add(opts)
	resp, err := get(d, path, p)

	if err != nil {
		return err
	}
	defer resp.Body.Close()
	n, err := findlastpage(resp.Header)
	if err != nil {
		return err
	}
	if err = init(resp.Body); err != nil {
		return err
	}

	for page := 2; page < n; page++ {
		p := params{
			"page":     {strconv.Itoa(page)},
			"per_page": {strconv.Itoa(perpage)},
		}
		p.Add(opts)
		resp, err = get(d, path, p)
		if err != nil {
			return err
		}
		if err = init(resp.Body); err != nil {
			resp.Body.Close()
			return err
		}
		resp.Body.Close()
	}
	return nil
}

var (
	resourceRegex = regexp.MustCompile(`<(.*?)>; rel="(.*?)"`)
	lastpageRegex = regexp.MustCompile(`.*<(.*)[\?&]page=([0-9]*).*>; rel="last"`)
)

func findlastpage(header http.Header) (int, error) {
	links := header.Get("Link")
	if links == "" {
		return -1, errs.New("no links found in the request header")
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

type linkedResource struct {
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
