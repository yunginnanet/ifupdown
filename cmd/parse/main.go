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
	ifaces := iface.NewMultiParser()
	switch {
	case len(os.Args) < 2:
		buf := &bytes.Buffer{}
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
		n, err := ifaces.Write(buf.Bytes())
		if err != nil {
			panic(err)
		}
		if n != len(buf.Bytes()) {
			panic("short write")
		}
	default:
		dat, err := os.ReadFile(os.Args[1])
		if err != nil {
			panic(err)
		}
		n, err := ifaces.Write(dat)
		if err != nil {
			panic(err)
		}
		if n != len(dat) {
			panic("short write")
		}
	}
	imap, err := ifaces.Parse()
	if err != nil {
		println(err.Error())
		return
	}
	for _, netif := range imap {
		if netif == nil {
			continue
		}
		if err = netif.Validate(); err != nil {
			println(netif.Name + " skip due to error: " + err.Error())
			delete(imap, netif.Name)
			continue
		}
	}

	dat, err := json.MarshalIndent(imap, "", "\t")
	_, _ = os.Stdout.Write(dat)
}
