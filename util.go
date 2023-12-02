package ifupdown

import (
	"bytes"
	"strings"
	"sync"
)

// lol a bit redundant

type poolGroup struct {
	Buffers p[*bytes.Buffer]
	Strs    p[*strings.Builder]
}

type p[T interface{ Reset() }] interface {
	Get() T
	Put(T)
}

type swim[T interface{ Reset() }] struct {
	pool *sync.Pool
}

func newPool[T interface{ Reset() }](new func() T) swim[T] {
	return swim[T]{
		pool: &sync.Pool{New: func() interface{} { return new() }},
	}
}

func (s *swim[T]) Get() T {
	return s.pool.Get().(T)
}

func (s *swim[T]) Put(t T) {
	t.Reset()
	s.pool.Put(t)
}

type bufs struct {
	buffers *swim[*bytes.Buffer]
	strings *swim[*strings.Builder]
}

func newBufs() bufs {
	b := newPool[*bytes.Buffer](func() *bytes.Buffer { return &bytes.Buffer{} })
	s := newPool[*strings.Builder](func() *strings.Builder { return &strings.Builder{} })
	return bufs{
		buffers: &b,
		strings: &s,
	}
}

var bufPools = newBufs()

var pools = poolGroup{Buffers: bufPools.buffers, Strs: bufPools.strings}
