package lacp

import (
	"context"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
	"go.uber.org/mock/gomock"

	"github.com/openshift/pf-status-relay/pkg/interfaces"
	"github.com/openshift/pf-status-relay/pkg/lacp/pf"
)

var _ = Describe("LACP", func() {
	Context("Inspect", func() {
		var (
			ctrl        *gomock.Controller
			mockNetlink *interfaces.MockNetlink
			nics        *Nics
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			mockNetlink = interfaces.NewMockNetlink(ctrl)

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

		AfterEach(func() {
			ctrl.Finish()
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
