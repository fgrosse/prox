package prox

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseEnvFile", func() {
	It("should read and return all environment variables", func() {
		envFile := `
			NAMESPACE=production
			FOO_URL=file://$GOPATH/src/github.com/foo/bar
			ETCD_ENDPOINT=localhost:2379
			LOG=*:debug,xxx:info,sd:info,cache:info,db:info
		`

		vars, err := ParseEnvFile(strings.NewReader(envFile))
		Expect(err).NotTo(HaveOccurred())
		Expect(vars).To(Equal([]string{
			"NAMESPACE=production",
			"FOO_URL=file://$GOPATH/src/github.com/foo/bar",
			"ETCD_ENDPOINT=localhost:2379",
			"LOG=*:debug,xxx:info,sd:info,cache:info,db:info",
		}))
	})

	It("should ignore comments", func() {
		envFile := `
			NAMESPACE=production
			#FOO_URL=file://$GOPATH/src/github.com/foo/bar
			FOO=bar
		`

		vars, err := ParseEnvFile(strings.NewReader(envFile))
		Expect(err).NotTo(HaveOccurred())
		Expect(vars).To(Equal([]string{
			"NAMESPACE=production",
			"FOO=bar",
		}))
	})

	It("should ignore empty lines", func() {
		envFile := `
			NAMESPACE=production

			FOO=BAR


			BAZ=BLUP
		`

		vars, err := ParseEnvFile(strings.NewReader(envFile))
		Expect(err).NotTo(HaveOccurred())
		Expect(vars).To(Equal([]string{
			"NAMESPACE=production",
			"FOO=BAR",
			"BAZ=BLUP",
		}))
	})

	It("should trim each line", func() {
		envFile := strings.Join([]string{
			"NAMESPACE=production\t \t\t  ",
			"FOO=bar  \t",
		}, "\n")

		vars, err := ParseEnvFile(strings.NewReader(envFile))
		Expect(err).NotTo(HaveOccurred())
		Expect(vars).To(Equal([]string{
			"NAMESPACE=production",
			"FOO=bar",
		}))
	})
})
