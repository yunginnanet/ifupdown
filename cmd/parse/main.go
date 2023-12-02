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
	eth0 := &iface.NetworkInterface{}
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
		n, err := eth0.Write(buf.Bytes())
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
		n, err := eth0.Write(dat)
		if err != nil {
			panic(err)
		}
		if n != len(dat) {
			panic("short write")
		}
	}
	dat, err := json.MarshalIndent(eth0, "", "\t")
	if err != nil {
		panic(err)
	}
	if err = eth0.Validate(); err != nil {
		println(err.Error())
	}
	_, _ = os.Stdout.Write(dat)
}
