package prox

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseProxFile", func() {
	Describe("simple Proxfile", func() {
		content := `
processes:
  redis: bin/redis-server conf/redis.conf
  server:   "php -S localhost:8080 app/web/index.php"
  selenium: "java -jar /usr/local/bin/selenium-server-standalone.jar"
`

		It("should parse process from the content", func() {
			processes, err := ParseProxFile(strings.NewReader(content), Environment{})
			Expect(err).NotTo(HaveOccurred())
			Expect(processes).To(HaveLen(3))
			Expect(processes).To(ContainShellTask("redis", "bin/redis-server conf/redis.conf"))
			Expect(processes).To(ContainShellTask("server", "php -S localhost:8080 app/web/index.php"))
			Expect(processes).To(ContainShellTask("selenium", "java -jar /usr/local/bin/selenium-server-standalone.jar"))
		})

		Describe("setting environment variables", func() {
			It("should pass the given environment to all created shell tasks", func() {
				env := Environment{
					"FOO": "bar",
				}
				processes, err := ParseProxFile(strings.NewReader(content), env)
				Expect(err).NotTo(HaveOccurred())
				for _, p := range processes {
					Expect(p.Env).To(Equal(env))
				}
			})
		})
	})

	Describe("complex Proxfile", func() {
		content := `
processes:
  redis: redis-server
  my-app:
    script: "app run now"
    env:
      - FOO=bar
      - test=false
    format: json
    fields:
      message: msg
      level: level
    tags:
      error:
        color: red
        condition:
          field: level
          value: error
      info:
        color: green
        condition:
          field: level
          value: info
`

		It("should parse process from the content", func() {
			env := Environment{"test": "true", "hello": "world"}
			processes, err := ParseProxFile(strings.NewReader(content), env)
			Expect(err).NotTo(HaveOccurred())
			Expect(processes).To(HaveLen(2))
			Expect(processes).To(ContainElement(Process{
				Name:   "redis",
				Script: "redis-server",
				Env:    env,
			}))
			Expect(processes).To(ContainElement(Process{
				Name:   "my-app",
				Script: "app run now",
				Env: Environment{
					"FOO":   "bar",
					"test":  "false",
					"hello": "world",
				},
				StructuredOutput: StructuredOutput{
					Format:       "json",
					MessageField: "msg",
					LevelField:   "level",
					TaggingRules: []TaggingRule{
						{
							Tag:   "error",
							Field: "level",
							Value: "error",
						},
						{
							Tag:   "info",
							Field: "level",
							Value: "info",
						},
					},
					TagColors: map[string]string{
						"error": "red",
						"info":  "green",
					},
				},
			}))
		})
	})
})
