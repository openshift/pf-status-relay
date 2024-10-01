package config

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	AfterEach(func() {
		err := os.Unsetenv(pfStatusRelayInterfaces)
		Expect(err).NotTo(HaveOccurred())

		err = os.Unsetenv(pfStatusRelayPollingInterval)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("ReadConfig", func() {
		It("should correctly read the env vars", func() {
			err := os.Setenv(pfStatusRelayInterfaces, "eth0,eth1")
			Expect(err).NotTo(HaveOccurred())

			err = os.Setenv(pfStatusRelayPollingInterval, "100")
			Expect(err).NotTo(HaveOccurred())

			// Call the function under test.
			c, err := ReadConfig()
			Expect(err).NotTo(HaveOccurred())

			// Validate the results.
			Expect(c.Interfaces).To(Equal([]string{"eth0", "eth1"}))
			Expect(c.PollingInterval).To(Equal(100))
		})

		It("should correctly use default value for pollingInterval", func() {
			err := os.Setenv(pfStatusRelayInterfaces, "eth0,eth1")
			Expect(err).NotTo(HaveOccurred())

			// Call the function under test.
			c, err := ReadConfig()
			Expect(err).NotTo(HaveOccurred())

			// Validate the results.
			Expect(c.Interfaces).To(Equal([]string{"eth0", "eth1"}))
			Expect(c.PollingInterval).To(Equal(1000))
		})

		It("should return an error when the polling interval is smaller than 100", func() {
			err := os.Setenv(pfStatusRelayPollingInterval, "50")
			Expect(err).NotTo(HaveOccurred())

			// Call the function under test.
			_, err = ReadConfig()
			Expect(err).To(HaveOccurred())
		})

		It("should return an error when the polling interval is not a number", func() {
			err := os.Setenv(pfStatusRelayPollingInterval, "error")
			Expect(err).NotTo(HaveOccurred())

			// Call the function under test.
			_, err = ReadConfig()
			Expect(err).To(HaveOccurred())
		})

		It("should return an error when interfaces is emptyr", func() {
			err := os.Setenv(pfStatusRelayInterfaces, "")
			Expect(err).NotTo(HaveOccurred())

			// Call the function under test.
			_, err = ReadConfig()
			Expect(err).To(HaveOccurred())
		})

		It("should properly trim spaces", func() {
			err := os.Setenv(pfStatusRelayInterfaces, "eth0,     eth1")
			Expect(err).NotTo(HaveOccurred())

			// Call the function under test.
			c, err := ReadConfig()
			Expect(err).NotTo(HaveOccurred())

			// Validate the results.
			Expect(c.Interfaces).To(Equal([]string{"eth0", "eth1"}))
		})

		It("should remove empty interfaces", func() {
			err := os.Setenv(pfStatusRelayInterfaces, "eth0,   ")
			Expect(err).NotTo(HaveOccurred())

			// Call the function under test.
			c, err := ReadConfig()
			Expect(err).NotTo(HaveOccurred())

			// Validate the results.
			Expect(c.Interfaces).To(Equal([]string{"eth0"}))
		})
	})
})
