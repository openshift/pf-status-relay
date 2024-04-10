package pf

import (
	"fmt"
	"sync"

	"github.com/vishvananda/netlink"

	"github.com/mlguerrero12/pf-status-relay/pkg/interfaces"
	"github.com/mlguerrero12/pf-status-relay/pkg/log"
)

// PF contains information about the physical function as well as a context to manage lacp monitoring.
type PF struct {
	// Name is the name of the interface.
	Name string
	// Index is the index of the interface.
	Index int
	// OperState is the operational state of the interface.
	OperState netlink.LinkOperState
	// MasterIndex is the index of the bond interface.
	MasterIndex int

	sync.Mutex
	Ready bool

	ProtoState protoState

	Nl interfaces.Netlink
}

type protoState int

const (
	Up protoState = iota
	Down
	NoVfs
	Undefined
)

func (p *PF) Inspect() error {
	// Verify that link is up.
	if p.OperState != netlink.OperUp {
		return fmt.Errorf("link is not up")
	}

	// Verify that link has a master.
	if p.MasterIndex == 0 {
		return fmt.Errorf("no master interface found")
	}

	// Verify that bond runs in mode 802.3ad.
	bond, err := p.Nl.LinkByIndex(p.MasterIndex)
	if err != nil {
		return fmt.Errorf("failed to fetch master interface with index %d: %w", p.MasterIndex, err)
	}

	// Verify that bond has mode 802.3ad
	if bond.(*netlink.Bond).Mode != netlink.BOND_MODE_802_3AD {
		return fmt.Errorf("bond %s does not have mode 802.3ad", bond.Attrs().Name)
	}

	return nil
}

// Update updates the info of the PF when there is an operational state change.
func (p *PF) Update() (bool, error) {
	// Fetch link again. Do not use attrs from subscribe since it might be obsolete.
	link, err := p.Nl.LinkByIndex(p.Index)
	if err != nil {
		return false, err
	}

	log.Log.Debug("link state", "state", link.Attrs().OperState)

	if link.Attrs().OperState == p.OperState {
		log.Log.Debug("PF was not updated", "interface", link.Attrs().Name)
		return false, nil
	}

	p.Name = link.Attrs().Name
	p.Index = link.Attrs().Index
	p.OperState = link.Attrs().OperState
	p.MasterIndex = link.Attrs().MasterIndex

	log.Log.Info("PF was updated", "interface", p.Name, "operational state", p.OperState)

	return true, nil
}
