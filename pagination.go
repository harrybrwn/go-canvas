package canvas

import (
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"strconv"
	"sync"
)

func newPaginatedList(
	d doer,
	path string,
	init func(io.Reader) ([]interface{}, error),
) *paginated {
	return &paginated{
		do:   d,
		init: init,
		path: path,
		wg:   new(sync.WaitGroup),
		errs: make(chan error),
	}
}

type paginated struct {
	wg   *sync.WaitGroup
	errs chan error
	path string
	init func(io.Reader) ([]interface{}, error)
	do   doer
}

func (p *paginated) channel() <-chan interface{} {
	var objects = make(chan interface{})
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
	n := lastpage.page
	p.wg.Add(n)

	go func() {
		defer resp.Body.Close()
		list, err := p.init(resp.Body)
		if err != nil {
			p.errs <- err
			return
		}
		for _, o := range list {
			objects <- o
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
				objects <- o
			}
			p.wg.Done()
		}(int64(page), p.path)
	}
	go func() {
		p.wg.Wait()
		close(objects)
		close(p.errs)
	}()
	return objects
}

func filesInit(r io.Reader) ([]interface{}, error) {
	files := make([]*File, 0)
	if err := json.NewDecoder(r).Decode(&files); err != nil {
		return nil, err
	}
	objects := make([]interface{}, len(files))
	for i, f := range files {
		objects[i] = f
	}
	return objects, nil
}
