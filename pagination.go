package canvas

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"sync"
)

func newPaginatedList(
	d doer,
	path string,
	init func(io.Reader) ([]interface{}, error),
) *paginated {
	return &paginated{
		do:      d,
		init:    init,
		path:    path,
		wg:      new(sync.WaitGroup),
		objects: make(chan interface{}),
		errs:    make(chan error),
	}
}

type paginated struct {
	path string
	do   doer

	n       int
	objects chan interface{}
	errs    chan error

	wg   *sync.WaitGroup
	init func(io.Reader) ([]interface{}, error)
}

// returns <number of pages>, <response>
func (p *paginated) firstReq() (int, *http.Response) {
	resp, err := get(p.do, p.path, url.Values{
		"page": {"1"},
	})
	if err != nil {
		p.errs <- err
	}
	pages, err := newLinkedResource(resp)
	if err != nil {
		p.errs <- err
	}
	lastpage, ok := pages.links["last"]
	if !ok {
		p.errs <- errors.New("could not find last page")
	}
	p.n = lastpage.page
	return p.n, resp
}

func (p *paginated) channel() <-chan interface{} {
	n, resp := p.firstReq()
	p.wg.Add(n)

	go func() {
		defer resp.Body.Close()
		list, err := p.init(resp.Body)
		if err != nil {
			p.errs <- err
			return
		}
		for _, o := range list {
			p.objects <- o
		}
		p.wg.Done()
	}()
	for page := 2; page <= n; page++ {
		go func(page int64, path string) {
			resp, err := get(p.do, path, url.Values{
				"page": {strconv.FormatInt(page, 10)},
			})
			if err != nil {
				p.errs <- err
				return
			}
			defer resp.Body.Close()
			obs, err := p.init(resp.Body)
			if err != nil {
				p.errs <- err
				return
			}
			for _, o := range obs {
				p.objects <- o
			}
			p.wg.Done()
		}(int64(page), p.path)
	}
	go func() {
		p.wg.Wait()
		close(p.objects)
		close(p.errs)
	}()
	return p.objects
}

func (p *paginated) collect() ([]interface{}, error) {
	p.channel()
	collection := make([]interface{}, 0, p.n*10)
	for {
		select {
		case err := <-p.errs:
			if err != nil {
				return nil, err
			}
		case obj := <-p.objects:
			if obj == nil {
				return collection, nil
			}
			collection = append(collection, obj)
		}
	}
}

var resourceRegex = regexp.MustCompile(`<(.*?)>; rel="(.*?)"`)

func newLinkedResource(rsp *http.Response) (*linkedResource, error) {
	var err error
	resource := &linkedResource{
		resp:  rsp,
		links: map[string]*link{},
	}
	links := rsp.Header.Get("Link")
	parts := resourceRegex.FindAllStringSubmatch(links, -1)

	for _, part := range parts {
		resource.links[part[2]], err = newlink(part[1])
		if err != nil {
			return resource, err
		}
	}
	return resource, nil
}

type linkedResource struct {
	resp  *http.Response
	links map[string]*link
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
