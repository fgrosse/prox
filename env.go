package prox

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Environment is a set of key value pairs that are used to set environment
// variables for processes.
type Environment map[string]string

// ParseEnvFile reads environment variables that should be set on all processes
// from the ".env" file.
//
// The format of the ".env" file is expected to be a newline separated list of
// key=value pairs which represent the environment variables that should be used
// by all started processes. Trimmed lines which are empty or start with a "#"
// are ignored and can be used to add comments.
//
// All values are expanded using the Environment. Additionally values in the env
// file can use other values which have been defined in earlier lines above. If
// a value refers to an unknown variable then it is replaced with the empty
// string.
func (e Environment) ParseEnvFile(r io.Reader) error {
	s := bufio.NewScanner(r)
	var i int
	for s.Scan() {
		i++

		line := strings.TrimSpace(s.Text())
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		if !strings.ContainsRune(line, '=') {
			return fmt.Errorf(`line %d does not contain '='`, i)
		}

		line = e.Expand(line)
		e.Set(line)
	}

	return s.Err()
}

// SystemEnv creates a new Environment from the operating systems environment
// variables.
func SystemEnv() Environment {
	return NewEnv(os.Environ())
}

// NewEnv creates a new Environment and immediately sets all given key=value
// pairs.
func NewEnv(values []string) Environment {
	env := Environment{}
	env.SetAll(values)
	return env
}

// Get retrieves the key from the Environment or returns the given default value
// if that key was not set
func (e Environment) Get(key, defaultValue string) string {
	if value, ok := e[key]; ok {
		return value
	}

	return defaultValue
}

// Set splits the input string at the first "=" character (if any) and sets the
// resulting key and value on e.
func (e Environment) Set(s string) {
	parts := strings.SplitN(s, "=", 2)
	if len(parts) == 1 {
		parts[1] = ""
	}

	parts[1] = strings.TrimSpace(parts[1])
	parts[1] = strings.TrimFunc(parts[1], func(r rune) bool {
		return r == '"' || r == '\''
	})

	e[parts[0]] = parts[1]
}

// SetAll assigns a list of key=value pairs on e.
func (e Environment) SetAll(vars []string) {
	for _, v := range vars {
		e.Set(v)
	}
}

// List returns all variables of e as a list of key=value pairs.
func (e Environment) List() []string {
	vars := make([]string, 0, len(e))
	for key, value := range e {
		vars = append(vars, fmt.Sprintf("%s=%s", key, value))
	}
	return vars
}

// Expand replaces ${var} or $var in the input string with the corresponding
// values of e. If the variable is not found in e then an empty string is
// returned.
func (e Environment) Expand(input string) string {
	return os.Expand(input, func(key string) string {
		return e[key]
	})
}
