package prox

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

// ParseEnvFile reads environment variables that should be set on all processes
// from the ".env" file and returns them as list of strings in "key=value"
// format.
func ParseEnvFile(r io.Reader) ([]string, error) { // TODO: return Environment?
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("could not read .env content: %s", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")

	var vars []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		vars = append(vars, line)
	}

	return vars, nil
}

type Environment map[string]string

func SystemEnv() Environment {
	return NewEnv(os.Environ())
}

func NewEnv(values []string) Environment {
	env := Environment{}
	env.SetAll(values)
	return env
}

func (e Environment) SetAll(vars []string) {
	for _, v := range vars {
		e.Set(v)
	}
}

func (e Environment) Set(v string) {
	parts := strings.SplitN(v, "=", 2)
	if len(parts) == 1 {
		parts[1] = ""
	}

	e[parts[0]] = parts[1]
}

func (e Environment) List() []string {
	vars := make([]string, len(e))
	for key, value := range e {
		vars = append(vars, fmt.Sprintf("%s=%s", key, value))
	}
	return vars
}

func (e Environment) Expand(input string) string {
	return os.Expand(input, func(key string) string {
		v, ok := e[key]
		if !ok {
			return key
		}
		return v
	})
}
