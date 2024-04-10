package interfaces

import "github.com/vishvananda/netlink"

type Netlink interface {
	LinkByIndex(int) (netlink.Link, error)
	LinkByName(string) (netlink.Link, error)
	LinkSetVfState(netlink.Link, int, uint32) error
}
