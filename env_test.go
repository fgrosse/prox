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

		env, err := ParseEnvFile(strings.NewReader(envFile))
		Expect(err).NotTo(HaveOccurred())
		Expect(env).To(Equal(Environment{
			"NAMESPACE":     "production",
			"FOO_URL":       "file://$GOPATH/src/github.com/foo/bar",
			"ETCD_ENDPOINT": "localhost:2379",
			"LOG":           "*:debug,xxx:info,sd:info,cache:info,db:info",
		}))
	})

	It("should ignore comments", func() {
		envFile := `
			NAMESPACE=production
			#FOO_URL=file://$GOPATH/src/github.com/foo/bar
			FOO=bar
		`

		env, err := ParseEnvFile(strings.NewReader(envFile))
		Expect(err).NotTo(HaveOccurred())
		Expect(env).To(Equal(Environment{
			"NAMESPACE": "production",
			"FOO":       "bar",
		}))
	})

	It("should ignore empty lines", func() {
		envFile := `
			NAMESPACE=production

			FOO=BAR


			BAZ=BLUP
		`

		env, err := ParseEnvFile(strings.NewReader(envFile))
		Expect(err).NotTo(HaveOccurred())
		Expect(env).To(Equal(Environment{
			"NAMESPACE": "production",
			"FOO":       "BAR",
			"BAZ":       "BLUP",
		}))
	})

	It("should trim each line", func() {
		envFile := strings.Join([]string{
			"NAMESPACE=production\t \t\t  ",
			"FOO=bar  \t",
		}, "\n")

		env, err := ParseEnvFile(strings.NewReader(envFile))
		Expect(err).NotTo(HaveOccurred())
		Expect(env).To(Equal(Environment{
			"NAMESPACE": "production",
			"FOO":       "bar",
		}))
	})
})

var _ = Describe("Environment", func() {
	Describe("Merge", func() {
		It("should add all variables from the other env", func() {
			env := Environment{"FOO": "bar"}
			env = env.Merge(Environment{"XXX": "yyy"})
			Expect(env).To(Equal(Environment{
				"FOO": "bar",
				"XXX": "yyy",
			}))
		})

		It("should not overwrite any existing variables", func() {
			env := Environment{"FOO": "bar"}
			env = env.Merge(Environment{"FOO": "baz"})
			Expect(env).To(Equal(Environment{"FOO": "bar"}))
		})
	})
})
