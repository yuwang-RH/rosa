package machinepools

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestIdp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MachinePools Suite")
}
