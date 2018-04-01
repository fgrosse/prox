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

			Expect(processes).To(ContainElement(Process{
				Name:             "redis",
				Script:           "bin/redis-server conf/redis.conf",
				Env:              Environment{},
				StructuredOutput: DefaultStructuredOutput,
			}))
			Expect(processes).To(ContainElement(Process{
				Name:             "server",
				Script:           "php -S localhost:8080 app/web/index.php",
				Env:              Environment{},
				StructuredOutput: DefaultStructuredOutput,
			}))
			Expect(processes).To(ContainElement(Process{
				Name:             "selenium",
				Script:           "java -jar /usr/local/bin/selenium-server-standalone.jar",
				Env:              Environment{},
				StructuredOutput: DefaultStructuredOutput,
			}))
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
      message: MESS
      level: level
    tags:
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
				Name:             "redis",
				Script:           "redis-server",
				Env:              env,
				StructuredOutput: DefaultStructuredOutput,
			}))

			conf := DefaultStructuredOutput
			conf.Format = "json"
			conf.MessageField = "MESS"
			conf.LevelField = "level"
			conf.TaggingRules = append(conf.TaggingRules, TaggingRule{
				Tag:   "info",
				Field: "level",
				Value: "info",
			})
			conf.TagColors["info"] = "green"
			Expect(processes).To(ContainElement(Process{
				Name:   "my-app",
				Script: "app run now",
				Env: Environment{
					"FOO":   "bar",
					"test":  "false",
					"hello": "world",
				},
				StructuredOutput: conf,
			}))
		})
	})
})
