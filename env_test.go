package prox

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Environment", func() {
	Describe("ParseEnvFile", func() {
		It("should read and return all environment variables", func() {
			envFile := `
				NAMESPACE=production
				FOO_URL=file://home/fgrosse/src/github.com/foo/bar
				ETCD_ENDPOINT=localhost:2379
				LOG=*:debug,xxx:info,sd:info,cache:info,db:info
			`

			env := Environment{}
			err := env.ParseEnvFile(strings.NewReader(envFile))
			Expect(err).NotTo(HaveOccurred())
			Expect(env).To(Equal(Environment{
				"NAMESPACE":     "production",
				"FOO_URL":       "file://home/fgrosse/src/github.com/foo/bar",
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

			env := Environment{}
			err := env.ParseEnvFile(strings.NewReader(envFile))
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

			env := Environment{}
			err := env.ParseEnvFile(strings.NewReader(envFile))
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

			env := Environment{}
			err := env.ParseEnvFile(strings.NewReader(envFile))
			Expect(err).NotTo(HaveOccurred())
			Expect(env).To(Equal(Environment{
				"NAMESPACE": "production",
				"FOO":       "bar",
			}))
		})

		It("should not remove escaped \\n strings at the end", func() {
			envFile := strings.Join([]string{
				`CASE_1=wtf1\n`,
				`CASE_2=wtf2\n\r`,
				`CASE_3=wtf2\r\n`,
				`CASE_4=wtf2\r`,
			}, "\n")

			env := Environment{}
			err := env.ParseEnvFile(strings.NewReader(envFile))
			Expect(err).NotTo(HaveOccurred())
			Expect(env).To(Equal(Environment{
				"CASE_1": `wtf1\n`,
				"CASE_2": `wtf2\n\r`,
				"CASE_3": `wtf2\r\n`,
				"CASE_4": `wtf2\r`,
			}))
		})

		It("should support spaces in values", func() {
			envFile := strings.Join([]string{
				"FOO=some value that contains spaces",
				"BAR=   spaces at the beginning or end shall be trimmed   ", // TODO? really? what if this is required?
			}, "\n")

			env := Environment{}
			err := env.ParseEnvFile(strings.NewReader(envFile))
			Expect(err).NotTo(HaveOccurred())
			Expect(env).To(Equal(Environment{
				"FOO": "some value that contains spaces",
				"BAR": "spaces at the beginning or end shall be trimmed",
			}))
		})

		It("should expand environment variables", func() {
			envFile := `
				PROX_TEST_1=it $PROX_TEST
				PROX_TEST_2=$PROX_TEST_1 really
				PROX_TEST_3=Yay! $PROX_TEST_2 well
				PROX_TEST_4=Empty $VARIABLE will be removed
				PROX_TEST_5=Spaces are trimmed automatically     
				PROX_TEST_6="You can use quotes to preserve them     "'
			`

			env := Environment{"PROX_TEST": "works"}
			err := env.ParseEnvFile(strings.NewReader(envFile))
			Expect(err).NotTo(HaveOccurred())
			Expect(env).To(Equal(Environment{
				"PROX_TEST":   "works",
				"PROX_TEST_1": "it works",
				"PROX_TEST_2": "it works really",
				"PROX_TEST_3": "Yay! it works really well",
				"PROX_TEST_4": "Empty  will be removed",
				"PROX_TEST_5": "Spaces are trimmed automatically",
				"PROX_TEST_6": "You can use quotes to preserve them     ",
			}))
		})

		It("should return an error if env file is malformed", func() {
			env := Environment{}
			err := env.ParseEnvFile(strings.NewReader("FOOBAR"))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("List", func() {
		It("should return all variables as list of key=value pairs", func() {
			env := Environment{"FOO": "bar", "BAZ": "..."}
			ee := env.List()
			Expect(ee).To(HaveLen(2))
			Expect(ee).To(ContainElement("FOO=bar"))
			Expect(ee).To(ContainElement("BAZ=..."))
		})
	})
})
