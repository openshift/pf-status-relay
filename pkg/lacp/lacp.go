package lacp

import (
	"context"
	"sync"
	"time"

	"github.com/vishvananda/netlink"

	"github.com/mlguerrero12/pf-status-relay/pkg/interfaces"
	"github.com/mlguerrero12/pf-status-relay/pkg/lacp/flags"
	"github.com/mlguerrero12/pf-status-relay/pkg/lacp/pf"
	"github.com/mlguerrero12/pf-status-relay/pkg/log"
)

// Nics stores the PFs that are inspected.
type Nics struct {
	PFs             map[int]*pf.PF
	queue           <-chan int
	pollingInterval int
	nl              interfaces.Netlink
}

// New returns an Nics structure with interfaces that are found in the node.
func New(nics []string, queue <-chan int, pollingInterval int, nl interfaces.Netlink) Nics {
	i := Nics{
		PFs:             make(map[int]*pf.PF),
		queue:           queue,
		pollingInterval: pollingInterval,
		nl:              nl,
	}
	for _, name := range nics {
		link, err := i.nl.LinkByName(name)
		if err != nil {
			log.Log.Warn("failed to fetch interface", "interface", name, "error", err)
			continue
		}

		log.Log.Debug("adding interface", "interface", name)

		i.PFs[link.Attrs().Index] = &pf.PF{
			Name:        link.Attrs().Name,
			Index:       link.Attrs().Index,
			OperState:   link.Attrs().OperState,
			MasterIndex: link.Attrs().MasterIndex,

			ProtoState: pf.Undefined,
			Nl:         nl,
		}
	}

	return i
}

// Inspect inspects interfaces in order to proceed with monitoring.
func (i *Nics) Inspect(ctx context.Context, wg *sync.WaitGroup) {
	log.Log.Debug("LACP inspection and processing started")

	// Verify that PFs are ready to accept/receive LACPDU messages.
	for _, p := range i.PFs {
		err := p.Inspect()
		if err != nil {
			log.Log.Error("pf is not ready", "interface", p.Name, "error", err)
			continue
		}
		log.Log.Info("pf is ready", "interface", p.Name)
		p.Ready = true
	}

	// Process link changes.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case index := <-i.queue:
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
						p.Lock()
						log.Log.Error("pf is not ready", "interface", p.Name, "error", err)
						p.Ready = false
						p.Unlock()
					} else {
						p.Lock()
						log.Log.Info("pf is ready", "interface", p.Name)
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
func (i *Nics) Monitor(ctx context.Context, wg *sync.WaitGroup) {
	log.Log.Debug("LACP monitoring started")

	wg.Add(1)
	go func() {
		defer func() {
			wg.Done()
		}()

		ticker := time.NewTicker(time.Duration(i.pollingInterval) * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				var monitorWg sync.WaitGroup
				for _, p := range i.PFs {
					monitorWg.Add(1)
					go func(p *pf.PF) {
						defer monitorWg.Done()

						p.Lock()
						if !p.Ready {
							p.Unlock()
							return
						}
						p.Unlock()

						link, err := i.nl.LinkByIndex(p.Index)
						if err != nil {
							log.Log.Warn("failed to fetch interface", "interface", p.Name, "error", err)
							return
						}

						// Stop if interface has no VFs.
						vfs := link.Attrs().Vfs
						if len(vfs) == 0 {
							if p.ProtoState != pf.NoVfs {
								log.Log.Info("interface has no VFs", "interface", p.Name)
								p.ProtoState = pf.NoVfs
							}
							return
						}

						// Check lacp state.
						slave := link.Attrs().Slave
						if slave != nil {
							s, ok := slave.(*netlink.BondSlave)
							if !ok {
								log.Log.Error("interface does not have BondSlave type on Slave attribute", "interface", p.Name)
								return
							}

							if flags.IsProtocolUp(s) {
								if p.ProtoState != pf.Up {
									log.Log.Info("lacp is up", "interface", p.Name)
									p.ProtoState = pf.Up
								}

								if !flags.IsFastRate(s) {
									log.Log.Warn("partner is using slow lacp rate", "interface", p.Name)
								}

								// Bring to auto all VFs whose state is disable.
								for _, vf := range vfs {
									log.Log.Debug("vf info", "id", vf.ID, "state", vf.LinkState, "interface", p.Name)
									if vf.LinkState == netlink.VF_LINK_STATE_DISABLE {
										err = i.nl.LinkSetVfState(link, vf.ID, netlink.VF_LINK_STATE_AUTO)
										if err != nil {
											log.Log.Error("failed to set vf link state", "id", vf.ID, "interface", p.Name, "error", err)
										}
										log.Log.Info("vf link state was set", "id", vf.ID, "state", "auto", "interface", p.Name)
									}
								}
							} else {
								if p.ProtoState != pf.Down {
									log.Log.Info("lacp is down", "interface", p.Name)
									p.ProtoState = pf.Down
								}

								// Bring to disable all VFs whose state is auto.
								for _, vf := range vfs {
									log.Log.Debug("vf info", "id", vf.ID, "state", vf.LinkState, "interface", p.Name)
									if vf.LinkState == netlink.VF_LINK_STATE_AUTO {
										err = i.nl.LinkSetVfState(link, vf.ID, netlink.VF_LINK_STATE_DISABLE)
										if err != nil {
											log.Log.Error("failed to set vf link state", "id", vf.ID, "interface", p.Name, "error", err)
										}
										log.Log.Info("vf link state was set", "id", vf.ID, "state", "disable", "interface", p.Name)
									}
								}
							}
						} else {
							log.Log.Error("interface has no slave attribute", "interface", p.Name)
						}
					}(p)

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
func (i *Nics) Indexes() []int {
	indexes := make([]int, 0, len(i.PFs))
	for index := range i.PFs {
		indexes = append(indexes, index)
	}

	return indexes
}
