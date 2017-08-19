package prox

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

// ParseEnvFile reads environment variables that should be set on all processes
// from the ".env" file and returns them as list of strings in "key=value"
// format.
func ParseEnvFile(r io.Reader) ([]string, error) {
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
