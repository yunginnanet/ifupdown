package ifupdown

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"git.tcp.direct/kayos/common/pool"
)

type Interfaces map[string]*NetworkInterface

func (i Interfaces) buf() *pool.Buffer {
	buf := pools.Buffers.Get()
	for _, iface := range i {
		err := iface.write(func(s string) { buf.MustWrite([]byte(s)) })
		if err != nil && !errors.Is(err, io.EOF) {
			panic(err)
		}
		buf.MustWrite([]byte("\n"))
	}
	return buf
}

func (i Interfaces) Read(p []byte) (int, error) {
	buf := i.buf()
	defer pools.Buffers.MustPut(buf)
	return buf.Read(p)
}

func (i Interfaces) String() string {
	buf := i.buf()
	defer pools.Buffers.MustPut(buf)
	return buf.String()
}

func (i Interfaces) UnmarshalJSON(data []byte) error {
	var ifaces map[string]*NetworkInterface
	if err := json.Unmarshal(data, &ifaces); err != nil {
		return err
	}
	for name, iface := range ifaces {
		iface.Name = name
		iface.allocated = true
		i[name] = iface
	}

	return nil
}

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
