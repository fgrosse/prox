package prox

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/matchers"
	"github.com/onsi/gomega/types"
)

var _ = Describe("ParseProcFile", func() {
	Describe("simple Procfile", func() {
		content := `
			redis: bin/redis-server conf/redis.conf
			server: php -S localhost:8080 app/web/index.php
			selenium: java -jar /usr/local/bin/selenium-server-standalone.jar
		`

		It("should parse shellProcess from the content", func() {
			processes, err := ParseProcFile(strings.NewReader(content), Environment{})
			Expect(err).NotTo(HaveOccurred())
			Expect(processes).To(HaveLen(3))
			Expect(processes).To(ContainShellTask("redis", "bin/redis-server conf/redis.conf"))
			Expect(processes).To(ContainShellTask("server", "php -S localhost:8080 app/web/index.php"))
			Expect(processes).To(ContainShellTask("selenium", "java -jar /usr/local/bin/selenium-server-standalone.jar"))
		})

		It("should ignore empty lines", func() {
			content = content + "\n\n\nfoo: test"
			processes, err := ParseProcFile(strings.NewReader(content), Environment{})
			Expect(err).NotTo(HaveOccurred())
			Expect(processes).To(HaveLen(4))
			Expect(processes).To(ContainShellTask("foo", "test"))
		})

		Describe("setting environment variables", func() {
			It("should pass the given environment to all created shell tasks", func() {
				env := Environment{
					"FOO": "bar",
				}
				processes, err := ParseProcFile(strings.NewReader(content), env)
				Expect(err).NotTo(HaveOccurred())
				for _, p := range processes {
					p, ok := p.(*shellProcess)
					Expect(ok).To(BeTrue())
					Expect(p.env).To(Equal(env))
				}
			})
		})
	})
})

func ContainShellTask(name, commandLine string) types.GomegaMatcher {
	return &matchers.ContainElementMatcher{
		Element: NewShellProcess(name, commandLine, Environment{}),
	}
}
