package config

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var oldPath string

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

var _ = BeforeSuite(func() {
	oldPath = path
})

var _ = AfterSuite(func() {
	path = oldPath
})
