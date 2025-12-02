package lacp

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
	"go.uber.org/mock/gomock"

	"github.com/openshift/pf-status-relay/pkg/interfaces"
	"github.com/openshift/pf-status-relay/pkg/lacp/pf"
	"github.com/openshift/pf-status-relay/pkg/log"
)

var _ = Describe("LACP", func() {
	var (
		logBuf         bytes.Buffer
		originalLogger *slog.Logger
		ctrl           *gomock.Controller
		mockNetlink    *interfaces.MockNetlink
		nics           *Nics
	)

	BeforeEach(func() {
		logBuf.Reset()
		originalLogger = log.Log
		log.Log = slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey { // do not print "time"
					return slog.Attr{}
				}
				return a
			},
		}))

		ctrl = gomock.NewController(GinkgoT())
		mockNetlink = interfaces.NewMockNetlink(ctrl)
	})

	AfterEach(func() {
		log.Log = originalLogger
		ctrl.Finish()
	})

	Context("Monitor", func() {
		BeforeEach(func() {
			nics = &Nics{
				PFs: map[int]*pf.PF{
					1: {
						Name:        "test",
						Index:       1,
						OperState:   netlink.OperUp,
						MasterIndex: 2,
						Ready:       true,
						ProtoState:  pf.Undefined,
						Nl:          mockNetlink,
					},
				},
				nl:              mockNetlink,
				pollingInterval: 100,
			}
		})

		Context("when VFs are created dynamically", func() {
			It("should detect and log VF creation", func() {
				bondSlave := &netlink.BondSlave{
					State:                  netlink.BondStateActive,
					MiiStatus:              netlink.BondLinkUp,
					LinkFailureCount:       0,
					PermHardwareAddr:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
					QueueId:                0,
					AggregatorId:           1,
					AdActorOperPortState:   60,
					AdPartnerOperPortState: 60,
				}

				linkWithoutVfs := &netlink.Dummy{
					LinkAttrs: netlink.LinkAttrs{
						Index: 1,
						Name:  "test",
						Vfs:   []netlink.VfInfo{},
						Slave: bondSlave,
					},
				}
				mockNetlink.EXPECT().LinkByIndex(1).Return(linkWithoutVfs, nil).Times(1)

				linkWithVfs := &netlink.Dummy{
					LinkAttrs: netlink.LinkAttrs{
						Index: 1,
						Name:  "test",
						Vfs: []netlink.VfInfo{
							{ID: 0, LinkState: netlink.VF_LINK_STATE_AUTO},
							{ID: 1, LinkState: netlink.VF_LINK_STATE_AUTO},
						},
						Slave: bondSlave,
					},
				}
				mockNetlink.EXPECT().LinkByIndex(1).Return(linkWithVfs, nil).AnyTimes()


				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				wg := &sync.WaitGroup{}
				nics.Monitor(ctx, wg)

				Eventually(func() bool {
					return nics.PFs[1].ProtoState == pf.NoVfs
				}, "1s", "50ms").Should(BeTrue())


				Eventually(func() bool {
					return nics.PFs[1].ProtoState != pf.NoVfs
				}, "1s", "50ms").Should(BeTrue())

				cancel()
				wg.Wait()

				wantLogs := `{"level":"INFO","msg":"pf has no VFs","interface":"test"}
{"level":"INFO","msg":"VFs detected on interface","interface":"test","count":2}
{"level":"INFO","msg":"lacp is up","interface":"test"}
{"level":"WARN","msg":"pf is using slow lacp rate","interface":"test"}
`

				if diff := cmp.Diff(wantLogs, logBuf.String()); diff != "" {
					Fail("Log mismatch (-want +got):\n" + diff)
				}
			})
		})
	})

	Context("Inspect", func() {
		BeforeEach(func() {
			nics = &Nics{
				PFs: map[int]*pf.PF{
					1: {
						Name:        "test",
						Index:       1,
						OperState:   netlink.OperUp,
						MasterIndex: 2,
						Nl:          mockNetlink,
					},
				},
				nl: mockNetlink,
			}
		})

		Context("when the PF is ready", func() {
			It("should set PF initially as ready and update to not ready when PF changes", func() {
				index := 1
				By("checking that the PF is ready")
				mockNetlink.EXPECT().LinkByIndex(gomock.Any()).Return(&netlink.Bond{Mode: netlink.BOND_MODE_802_3AD}, nil)

				ctx, cancel := context.WithCancel(context.Background())
				queue := make(chan int, 1)
				nics.queue = queue

				wg := &sync.WaitGroup{}
				nics.Inspect(ctx, wg)
				Eventually(func() bool {
					return nics.PFs[1].Ready
				}, "2s", "1s").Should(BeTrue())

				By("checking that the PF is not ready after updating PF")
				mockNetlink.EXPECT().LinkByIndex(gomock.Any()).Return(&netlink.Dummy{
					LinkAttrs: netlink.LinkAttrs{
						Name:        "test",
						Index:       index,
						OperState:   netlink.OperDown,
						MasterIndex: 0,
					},
				}, nil)

				queue <- index

				Eventually(func() bool {
					return nics.PFs[1].Ready
				}, "2s", "1s").Should(BeFalse())

				cancel()
				done := make(chan struct{})
				go func() {
					wg.Wait()
					close(done)
				}()

				select {
				case <-done:
					// wg.Wait() finished within 1 second
				case <-time.After(1 * time.Second):
					// wg.Wait() did not finish within 1 second, fail the test
					Fail("wg.Wait() did not finish within 1 second")
				}
				wg.Wait()
			})
		})
		Context("when the PF is not ready", func() {
			It("should set PF initially as not ready and update to ready when PF changes", func() {
				index := 1
				By("checking that the PF is not ready")
				mockNetlink.EXPECT().LinkByIndex(gomock.Any()).Return(&netlink.Bond{Mode: netlink.BOND_MODE_BALANCE_RR}, nil)

				ctx, cancel := context.WithCancel(context.Background())
				queue := make(chan int, 1)
				nics.queue = queue

				wg := &sync.WaitGroup{}
				nics.Inspect(ctx, wg)
				Eventually(func() bool {
					return nics.PFs[1].Ready
				}, "2s", "1s").Should(BeFalse())

				By("checking that the PF is ready after updating PF")
				mockNetlink.EXPECT().LinkByIndex(gomock.Any()).Return(&netlink.Dummy{
					LinkAttrs: netlink.LinkAttrs{
						Name:        "test",
						Index:       index,
						OperState:   netlink.OperDown,
						MasterIndex: 2,
					},
				}, nil)

				queue <- index

				mockNetlink.EXPECT().LinkByIndex(gomock.Any()).Return(&netlink.Dummy{
					LinkAttrs: netlink.LinkAttrs{
						Name:        "test",
						Index:       index,
						OperState:   netlink.OperUp,
						MasterIndex: 2,
					},
				}, nil)
				mockNetlink.EXPECT().LinkByIndex(gomock.Any()).Return(&netlink.Bond{Mode: netlink.BOND_MODE_802_3AD}, nil)

				queue <- index

				Eventually(func() bool {
					return nics.PFs[1].Ready
				}, "2s", "1s").Should(BeTrue())

				cancel()
				done := make(chan struct{})
				go func() {
					wg.Wait()
					close(done)
				}()

				select {
				case <-done:
					// wg.Wait() finished within 1 second
				case <-time.After(1 * time.Second):
					// wg.Wait() did not finish within 1 second, fail the test
					Fail("wg.Wait() did not finish within 1 second")
				}
				wg.Wait()
			})
		})
	})
})
