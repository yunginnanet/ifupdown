# ifupdown

[![GoDoc](https://godoc.org/git.tcp.direct/kayos/ifupdown?status.svg)](https://godoc.org/git.tcp.direct/kayos/ifupdown)


golang library for working with network interfaces defined via [ifupdown](https://manpages.debian.org/jessie/ifupdown/interfaces.5.en.html).
commonly recognized as `/etc/network/interfaces`.

## library

### features

- [x] read+parse interfaces file
- [x] write interfaces file
- [x] validate interfaces file (basic)
- [ ] validate interfaces file (thorough)
- [x] translate interfaces file to JSON
- [x] translate JSON to interfaces file

## cmd

- `ifup2json` - translate interfaces file to JSON
- `json2ifup` - translate JSON to interfaces file

### example usage

<details>
<summary>ifup2json</summary>

`cat /etc/network/interfaces | ifup2json`

```json
{
	"eth2": {
		"name": "eth2",
		"auto": true,
		"address": "192.168.69.5",
		"netmask": "////AA==",
		"gateway": "192.168.69.1",
		"config": 3,
		"version": 1,
		"hooks": {
			"pre_up": [
				"echo yeet"
			],
			"post_down": [
				"echo yeeted"
			]
		}
	},
	"lo": {
		"name": "lo",
		"auto": true,
		"config": 1,
		"version": 1,
		"hooks": {}
	},
	"ns3": {
		"name": "ns3",
		"auto": true,
		"address": "10.9.0.6",
		"netmask": "////AAAAAAAAAAAAAAAAAA==",
		"config": 3,
		"version": 1,
		"hooks": {
			"pre_up": [
				"ip link add dev ns3 type wireguard"
			],
			"post_up": [
				"wg setconf ns3 /etc/wireguard/ns3.conf"
			]
		}
	}
}
```

</details>

<details>
<summary>json2ifup</summary>

`cat /etc/network/interfaces | ifup2json | json2ifup`

```
auto eth2
iface eth2 inet static
	address 192.168.69.5
	netmask 255.255.255.0
	gateway 192.168.69.1
	pre-up echo yeet
	post-down echo yeeted

auto lo
iface lo inet loopback

auto ns3
iface ns3 inet static
	address 10.9.0.6
	netmask 255.255.255.0
	pre-up ip link add dev ns3 type wireguard
	post-up wg setconf ns3 /etc/wireguard/ns3.conf
```

</details>
