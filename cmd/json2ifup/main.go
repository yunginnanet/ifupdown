package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"

	iface "git.tcp.direct/kayos/ifupdown"
)

func main() {
	var ifaces = make(iface.Interfaces)
	buf := &bytes.Buffer{}

	switch {
	case len(os.Args) < 2:
		var empty = 0
		for {
			n, err := buf.ReadFrom(os.Stdin)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				panic(err)
			}
			if n == 0 {
				empty++
			}
			if empty > 100 {
				break
			}
		}
	default:
		f, err := os.Open(os.Args[1])
		if err != nil {
			println(err.Error())
			return
		}
		_, _ = buf.ReadFrom(f)
	}
	err := json.Unmarshal(buf.Bytes(), &ifaces)
	if err != nil {
		println(err.Error())
		// println("input received: " + string(buf.Bytes()))
		return
	}

	for name, netif := range ifaces {
		if netif == nil {
			delete(ifaces, name)
			continue
		}
		if netif.Name != name {
			panic("name mismatch")
		}
		if err = netif.Validate(); err != nil {
			println(name + ": skip due to error: " + err.Error())
			delete(ifaces, name)
			continue
		}
	}

	if _, err = os.Stdout.WriteString(ifaces.String()); err != nil {
		panic(err)
	}
}
