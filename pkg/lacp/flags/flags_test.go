package flags

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
)

var _ = Describe("Flags", func() {
	Describe("isOperational", func() {
		It("should return true when flags are Distributing, Collecting, Synchronization, and Aggregation", func() {
			f := flags(Distributing | Collecting | Synchronization | Aggregation)
			Expect(f.isOperational()).To(BeTrue())
		})

		It("should return false when flags are Expired", func() {
			f := flags(Expired)
			Expect(f.isOperational()).To(BeFalse())
		})

		It("should return false when flags are Defaulted", func() {
			f := flags(Defaulted)
			Expect(f.isOperational()).To(BeFalse())
		})

		It("should return false when flags are missing any of Distributing, Collecting, Synchronization, or Aggregation", func() {
			f := flags(Distributing | Collecting)
			Expect(f.isOperational()).To(BeFalse())
		})
	})

	Describe("IsFastRate", func() {
		It("should return true when Timeout flag is set", func() {
			slave := &netlink.BondSlave{
				AdPartnerOperPortState: Timeout,
			}
			Expect(IsFastRate(slave)).To(BeTrue())
		})

		It("should return false when Timeout flag is not set", func() {
			slave := &netlink.BondSlave{
				AdPartnerOperPortState: Activity,
			}
			Expect(IsFastRate(slave)).To(BeFalse())
		})
	})
})
