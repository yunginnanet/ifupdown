package iface

import (
	"bufio"
	"fmt"
	"strings"
	"sync"

	"git.tcp.direct/kayos/common/pool"
)

type Interfaces map[string]*NetworkInterface

type poolGroup struct {
	Buffers pool.BufferFactory
	Strs    pool.StringFactory
}

var pools = poolGroup{Buffers: pool.NewBufferFactory(), Strs: pool.NewStringFactory()}

type MultiParser struct {
	Interfaces map[string]*NetworkInterface
	Errs       []error
	buf        []byte
	mu         *sync.Mutex
}

func NewMultiParser() *MultiParser {
	return &MultiParser{
		Interfaces: make(Interfaces),
		Errs:       make([]error, 0),
		buf:        make([]byte, 0),
		mu:         &sync.Mutex{},
	}
}

func (p *MultiParser) Write(data []byte) (int, error) {
	p.mu.Lock()
	p.buf = append(p.buf, data...)
	p.mu.Unlock()
	return len(data), nil
}

func (p *MultiParser) Parse() (Interfaces, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	scanner := bufio.NewScanner(strings.NewReader(string(p.buf)))

	index := 0
	currentIfaceName := ""

	buf := pools.Buffers.Get()
	defer pools.Buffers.MustPut(buf)

	flush := func(name string) (*NetworkInterface, bool) {
		if len(buf.Bytes()) == 0 {
			return nil, false
		}
		defer buf.MustReset()
		newIface := NewNetworkInterface(name)
		if _, err := buf.WriteTo(newIface); err != nil {
			p.Errs = append(p.Errs, err)
			return nil, false
		}
		p.Interfaces[newIface.Name] = newIface
		return newIface, true
	}

	w := func(s string) {
		_, _ = buf.WriteString(s)
		_ = buf.WriteByte('\n')
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		upForChange := currentIfaceName == "" ||
			(currentIfaceName != "" && !strings.Contains(line, currentIfaceName)) ||
			index == 0
		startDetected := len(strings.Fields(line)) > 1 &&
			(strings.HasPrefix(line, "auto") ||
				strings.HasPrefix(line, "allow-") ||
				strings.HasPrefix(line, "iface"))

		switch {
		case line == "", strings.HasPrefix(line, "#"):
			continue
		case upForChange && startDetected:
			newName := strings.Fields(line)[1]
			if ifa, ok := flush(newName); ok {
				currentIfaceName = ifa.Name
				index++
			}
			w(line)
			continue
		default:
			w(line)
		}
	}

	if len(buf.Bytes()) > 0 {
		flush("unknown")
	}
	var multiErr error
	for _, err := range p.Errs {
		switch {
		case err == nil:
			continue
		case multiErr == nil:
			multiErr = err
		default:
			multiErr = fmt.Errorf("%w, %w", multiErr, err)
		}
	}
	return p.Interfaces, multiErr
}
