package test

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rjeczalik/notify"
)

type visited string
type end string

func mark(s string) notify.WalkNodeFunc {
	return func(nd notify.Node, last bool) (err error) {
		if nd.Name != sep {
			if last {
				dir, ok := nd.Parent[nd.Name].(map[string]interface{})
				if !ok {
					dir = make(map[string]interface{})
					nd.Parent[nd.Name] = dir
				}
				dir[""] = end(s)
			}
			nd.Parent[""] = visited(s)
		} else {
			if last {
				nd.Parent[""] = end(s)
			} else {
				nd.Parent[""] = visited(s)
			}
		}
		return
	}
}

func sendlast(c chan<- notify.Node) notify.WalkNodeFunc {
	return func(nd notify.Node, last bool) error {
		if last {
			c <- nd
		}
		return nil
	}
}

// WalkCase TODO
type WalkCase struct {
	C string
	W []string
}

type p struct {
	t *testing.T
	w *notify.WatchPointTree
}

// P TODO
func P(t *testing.T) *p {
	w := notify.NewWatchPointTree(nil)
	w.FS = FS
	return &p{t: t, w: w}
}

// Close TODO
func (p *p) Close() error {
	return p.w.Close()
}

func (p *p) expectmark(it map[string]interface{}, mark string, dirs []string) {
	v, ok := it[""]
	if !ok {
		p.t.Errorf("dir has no mark (mark=%q)", mark)
	}
	typ := reflect.TypeOf(visited(""))
	if len(dirs) == 0 {
		typ = reflect.TypeOf(end(""))
	}
	if got := reflect.TypeOf(v); got != typ {
		p.t.Errorf("want typeof(v)=%v; got %v (mark=%q)", typ, got, mark)
	}
	if reflect.ValueOf(v).String() != mark {
		p.t.Errorf("want v=%v; got %v (mark=%q)", mark, v, mark)
	}
	delete(it, "")
	for i, dir := range dirs {
		v, ok := it[dir]
		if !ok {
			p.t.Errorf("dir not found (mark=%q, i=%d)", mark, i)
			break
		}
		if it, ok = v.(map[string]interface{}); !ok {
			p.t.Errorf("want typeof(v)=map[string]interface; got %+v (mark=%q, i=%d)",
				v, mark, i)
			break
		}
		if v, ok = it[""]; !ok {
			p.t.Errorf("dir has no mark (mark=%q, i=%d)", mark, i)
			break
		}
		typ := reflect.TypeOf(visited(""))
		if i == len(dirs)-1 {
			typ = reflect.TypeOf(end(""))
		}
		if got := reflect.TypeOf(v); got != typ {
			p.t.Errorf("want typeof(v)=%v; got %v (mark=%q, i=%d)", typ, got, mark, i)
			continue
		}
		if reflect.ValueOf(v).String() != mark {
			p.t.Errorf("want v=%v; got %v (mark=%q, i=%d)", mark, v, mark, i)
			continue
		}
		delete(it, "") // remove visitation mark
	}
}

// Test for dangling marks - if a mark is present, WalkPoint went somewhere
// it shouldn't.
func (p *p) expectnomark() error {
	return p.w.BFS(func(v interface{}) error {
		switch v.(type) {
		case map[string]interface{}:
			return nil
		default:
			return fmt.Errorf("want typeof(...)=map[string]interface{}; got %T=%v", v, v)
		}
	})
}

// ExpectWalk TODO
//
// For each test-case we're traversing path specified by a testcase's key
// over shared WatchPointTree and marking each directory using special empty
// key. The mark is simply the traversed path name. Each mark can be either
// of `visited` or `end` type. Only the last item in the path is marked with
// an `end` mark.
func (p *p) ExpectWalk(cases map[string][]string) {
	for path, dirs := range cases {
		path = filepath.Clean(filepath.FromSlash(path))
		if err := p.w.WalkNode(path, mark(path)); err != nil {
			p.t.Errorf("want err=nil; got %v (path=%q)", err, path)
			continue
		}
		p.expectmark(p.w.Root, path, dirs)
	}
	if err := p.expectnomark(); err != nil {
		p.t.Error(err)
	}
}

// ExpectWalkCwd TODO
func (p *p) ExpectWalkCwd(cases map[string]WalkCase) {
	p.t.Skip("probably to be deleted")
	for path, cas := range cases {
		path = filepath.Clean(filepath.FromSlash(path))
		cas.C = filepath.Clean(filepath.FromSlash(cas.C))
		c := make(chan notify.Node, 1)
		// Prepare - look up cwd Point by walking its subpath.
		if err := p.w.WalkNode(filepath.Join(cas.C, "test"), sendlast(c)); err != nil {
			p.t.Errorf("want err=nil; got %v (path=%q)", err, path)
			continue
		}
		select {
		case nd := <-c:
			p.w.Cwd.Parent, p.w.Cwd.Name = nd.Parent, cas.C
		default:
			p.t.Errorf("unable to find cwd Point (path=%q)", path)
		}
		// Actual test.
		if err := p.w.WalkNode(path, mark(path)); err != nil {
			p.t.Errorf("want err=nil; got %v (path=%q)", err, path)
			continue
		}
		p.expectmark(p.w.Cwd.Parent, path, cas.W)
	}
	if err := p.expectnomark(); err != nil {
		p.t.Error(err)
	}
}

// ExpectWalk TODO
func ExpectWalk(t *testing.T, cases map[string][]string) {
	p := P(t)
	defer p.Close()
	p.ExpectWalk(cases)
}

// ExpectWalkCwd TODO
func ExpectWalkCwd(t *testing.T, cases map[string]WalkCase) {
	p := P(t)
	defer p.Close()
	p.ExpectWalkCwd(cases)
}