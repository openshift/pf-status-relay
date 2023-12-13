package config

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	var tmpFile *os.File
	BeforeEach(func() {
		var err error

		// Create a temporary file and write some data to it.
		tmpFile, err = os.CreateTemp("", "example")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := os.Remove(tmpFile.Name())
		Expect(err).NotTo(HaveOccurred())
	})

	Context("ReadConfig", func() {
		It("should correctly read the config file", func() {
			text := []byte("interfaces: [\"eth0\", \"eth1\"]\npollingInterval: 100\n")
			_, err := tmpFile.Write(text)
			Expect(err).NotTo(HaveOccurred())
			err = tmpFile.Close()
			Expect(err).NotTo(HaveOccurred())

			// Temporarily replace the config path
			path = tmpFile.Name()

			// Call the function under test.
			c := ReadConfig()

			// Validate the results.
			Expect(c.Interfaces).To(Equal([]string{"eth0", "eth1"}))
			Expect(c.PollingInterval).To(Equal(100))
		})

		It("should correctly use default value for pollingInterval", func() {
			text := []byte("interfaces: [\"eth0\", \"eth1\"]\n")
			_, err := tmpFile.Write(text)
			Expect(err).NotTo(HaveOccurred())
			err = tmpFile.Close()
			Expect(err).NotTo(HaveOccurred())

			// Temporarily replace the config path
			path = tmpFile.Name()

			// Call the function under test.
			c := ReadConfig()

			// Validate the results.
			Expect(c.Interfaces).To(Equal([]string{"eth0", "eth1"}))
			Expect(c.PollingInterval).To(Equal(1000))
		})
	})
})
