package lacp

import (
	"context"
	"github.com/mlguerrero12/pf-status-relay/pkg/lacp/flags"
	"sync"
	"time"

	"github.com/vishvananda/netlink"

	"github.com/mlguerrero12/pf-status-relay/pkg/log"
)

// Interfaces stores the PFs that are inspected.
type Interfaces struct {
	PFs map[int]*PF
}

// New returns an Interfaces structure with interfaces that are found in the node.
func New(nics []string) Interfaces {
	i := Interfaces{PFs: make(map[int]*PF)}
	for _, name := range nics {
		link, err := netlink.LinkByName(name)
		if err != nil {
			log.Log.Warn("failed to fetch interface", "interface", name, "error", err)
			continue
		}

		log.Log.Debug("adding interface", "interface", name)

		i.PFs[link.Attrs().Index] = &PF{
			Name:        link.Attrs().Name,
			Index:       link.Attrs().Index,
			OperState:   link.Attrs().OperState,
			MasterIndex: link.Attrs().MasterIndex,

			protoState: Undefined,
		}
	}

	return i
}

// Inspect inspects interfaces in order to proceed with monitoring.
func (i *Interfaces) Inspect(ctx context.Context, queue <-chan int, wg *sync.WaitGroup) {
	log.Log.Debug("LACP inspection and processing started")

	// Verify that PFs are ready to accept/receive LACPDU messages.
	for _, p := range i.PFs {
		err := p.Inspect()
		if err != nil {
			log.Log.Error("interface not ready", "interface", p.Name, "error", err)
			continue
		}
		p.Ready = true
	}

	// Process link changes.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case index := <-queue:
				log.Log.Debug("processing event", "index", index)
				p := i.PFs[index]
				updated, err := p.Update()
				if err != nil {
					log.Log.Error("failed to update link", "interface", p.Name, "error", err)
					break
				}

				if updated {
					err = p.Inspect()
					if err != nil {
						log.Log.Error("interface not ready after update", "interface", p.Name, "error", err)
						p.Lock()
						p.Ready = false
						p.Unlock()
					} else {
						p.Lock()
						p.Ready = true
						p.Unlock()
					}
				}
			case <-ctx.Done():
				log.Log.Debug("ctx cancelled", "routine", "inspect")
				return
			}
		}
	}()
}

// Monitor monitors the LACP protocol on the interfaces.
func (i *Interfaces) Monitor(ctx context.Context, pollingInterval int, wg *sync.WaitGroup) {
	log.Log.Debug("LACP monitoring started")

	wg.Add(1)
	go func() {
		defer func() {
			wg.Done()
		}()

		for {
			select {
			case <-time.Tick(time.Duration(pollingInterval) * time.Millisecond):
				var monitorWg sync.WaitGroup
				for _, pf := range i.PFs {
					monitorWg.Add(1)
					go func(pf *PF) {
						defer monitorWg.Done()

						pf.Lock()
						if !pf.Ready {
							pf.Unlock()
							return
						}
						pf.Unlock()

						link, err := netlink.LinkByIndex(pf.Index)
						if err != nil {
							log.Log.Warn("failed to fetch interface", "interface", pf.Name, "error", err)
							return
						}

						// Stop if interface has no VFs.
						vfs := link.Attrs().Vfs
						if len(vfs) == 0 {
							if pf.protoState != NoVfs {
								log.Log.Info("interface has no VFs", "interface", pf.Name)
								pf.protoState = NoVfs
							}
							return
						}

						// Check lacp state.
						slave := link.Attrs().Slave
						if slave != nil {
							s, ok := slave.(*netlink.BondSlave)
							if !ok {
								log.Log.Error("interface does not have BondSlave type on Slave attribute", "interface", pf.Name)
								return
							}

							if flags.IsProtocolUp(s) {
								if pf.protoState != Up {
									log.Log.Info("lacp is up", "interface", pf.Name)
									pf.protoState = Up
								}

								if !flags.IsFastRate(s) {
									log.Log.Warn("partner is using slow lacp rate", "interface", pf.Name)
								}

								// Bring to auto all VFs whose state is disable.
								for _, vf := range vfs {
									log.Log.Debug("vf info", "id", vf.ID, "state", vf.LinkState, "interface", pf.Name)
									if vf.LinkState == netlink.VF_LINK_STATE_DISABLE {
										err = netlink.LinkSetVfState(link, vf.ID, netlink.VF_LINK_STATE_AUTO)
										if err != nil {
											log.Log.Error("failed to set vf link state", "id", vf.ID, "interace", pf.Name, "error", err)
										}
										log.Log.Info("vf link state was set", "id", vf.ID, "state", "auto", "interface", pf.Name)
									}
								}
							} else {
								if pf.protoState != Down {
									log.Log.Info("lacp is down", "interface", pf.Name)
									pf.protoState = Down
								}

								// Bring to disable all VFs whose state is auto.
								for _, vf := range vfs {
									log.Log.Debug("vf info", "id", vf.ID, "state", vf.LinkState, "interface", pf.Name)
									if vf.LinkState == netlink.VF_LINK_STATE_AUTO {
										err = netlink.LinkSetVfState(link, vf.ID, netlink.VF_LINK_STATE_DISABLE)
										if err != nil {
											log.Log.Error("failed to set vf link state", "id", vf.ID, "interface", pf.Name, "error", err)
										}
										log.Log.Info("vf link state was set", "id", vf.ID, "state", "disable", "interface", pf.Name)
									}
								}
							}
						} else {
							log.Log.Error("interface has no slave attribute", "interface", pf.Name)
						}
					}(pf)

					monitorWg.Wait()
				}
			case <-ctx.Done():
				log.Log.Debug("ctx cancelled", "routine", "monitor")
				return
			}
		}
	}()

}

// Indexes returns a list of indexes.
func (i *Interfaces) Indexes() []int {
	indexes := make([]int, 0, len(i.PFs))
	for index := range i.PFs {
		indexes = append(indexes, index)
	}

	return indexes
}
