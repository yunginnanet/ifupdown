package ifupdown

import "git.tcp.direct/kayos/common/pool"

type poolGroup struct {
	Buffers pool.BufferFactory
	Strs    pool.StringFactory
}

var pools = poolGroup{Buffers: pool.NewBufferFactory(), Strs: pool.NewStringFactory()}
