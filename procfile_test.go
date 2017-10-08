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

		It("should parse process from the content", func() {
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

		It("should ignore commented lines", func() {
			content := `
				#redis: bin/redis-server conf/redis.conf
				server: start_server.sh
			`
			processes, err := ParseProcFile(strings.NewReader(content), Environment{})
			Expect(err).NotTo(HaveOccurred())
			Expect(processes).To(HaveLen(1))
			Expect(processes).To(ContainShellTask("server", "start_server.sh"))
		})

		Describe("setting environment variables", func() {
			It("should pass the given environment to all created shell tasks", func() {
				env := Environment{
					"FOO": "bar",
				}
				processes, err := ParseProcFile(strings.NewReader(content), env)
				Expect(err).NotTo(HaveOccurred())
				for _, p := range processes {
					Expect(p.Env).To(Equal(env))
				}
			})
		})
	})
})

func ContainShellTask(name, commandLine string) types.GomegaMatcher {
	return &matchers.ContainElementMatcher{
		Element: Process{Name: name, Script: commandLine, Env: Environment{}},
	}
}
