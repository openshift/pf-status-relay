package pf

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
	"go.uber.org/mock/gomock"

	"github.com/openshift/pf-status-relay/pkg/interfaces"
)

var _ = Describe("PF", func() {
	Describe("Inspect", func() {
		var (
			ctrl        *gomock.Controller
			mockNetlink *interfaces.MockNetlink
			pf          *PF
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			mockNetlink = interfaces.NewMockNetlink(ctrl)

			pf = &PF{
				Name:  "test",
				Index: 1,
				Nl:    mockNetlink,
			}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		Context("when the link is up and has a master", func() {
			It("should not return an error", func() {
				pf.OperState = netlink.OperUp
				pf.MasterIndex = 2

				mockNetlink.EXPECT().LinkByIndex(gomock.Any()).Return(&netlink.Bond{Mode: netlink.BOND_MODE_802_3AD}, nil).Times(1)

				err := pf.Inspect()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the link is not up", func() {
			It("should return an error", func() {
				pf.OperState = netlink.OperDown
				err := pf.Inspect()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("link is not up"))
			})
		})

		Context("when there is no master interface", func() {
			It("should return an error", func() {
				pf.OperState = netlink.OperUp
				pf.MasterIndex = 0
				err := pf.Inspect()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("link has no master interface"))
			})
		})

		Context("when the bond does not have mode 802.3ad", func() {
			It("should return an error", func() {
				pf.OperState = netlink.OperUp
				pf.MasterIndex = 2

				mockNetlink.EXPECT().LinkByIndex(gomock.Any()).Return(&netlink.Bond{
					LinkAttrs: netlink.LinkAttrs{Name: "test"},
					Mode:      netlink.BOND_MODE_BALANCE_RR,
				}, nil).Times(1)
				err := pf.Inspect()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("bond test does not have mode 802.3ad"))
			})
		})

		Context("when the link state does not change", func() {
			It("should not update the PF", func() {
				pf.OperState = netlink.OperUp
				pf.MasterIndex = 2

				mockNetlink.EXPECT().LinkByIndex(gomock.Any()).Return(&netlink.Dummy{
					LinkAttrs: netlink.LinkAttrs{
						Name:        "test",
						Index:       1,
						OperState:   netlink.OperUp,
						MasterIndex: 2,
					},
				}, nil).Times(1)

				updated, err := pf.Update()
				Expect(err).NotTo(HaveOccurred())
				Expect(updated).To(BeFalse())
			})
		})

		Context("when the link state changes", func() {
			It("should update the PF", func() {
				pf.OperState = netlink.OperUp
				pf.MasterIndex = 2

				mockNetlink.EXPECT().LinkByIndex(gomock.Any()).Return(&netlink.Dummy{
					LinkAttrs: netlink.LinkAttrs{
						Name:        "test",
						Index:       1,
						OperState:   netlink.OperDown,
						MasterIndex: 0,
					},
				}, nil).Times(1)

				updated, err := pf.Update()
				Expect(err).NotTo(HaveOccurred())
				Expect(updated).To(BeTrue())
				Expect(int(pf.OperState)).To(Equal(netlink.OperDown))
				Expect(pf.MasterIndex).To(Equal(0))
			})
		})
	})
})
