package prox

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

func ParseProcFile(reader io.Reader, env Environment) ([]Process, error) {
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read Procfile content: %s", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var processes []Process
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		lineParts := strings.SplitN(line, ":", 2)
		if len(lineParts) < 2 {
			return processes, fmt.Errorf("invalid Procfile format at line %d: %s", i+1, line)
		}

		name := strings.TrimSpace(lineParts[0])
		script := strings.TrimSpace(lineParts[1])

		processes = append(processes, NewShellProcess(name, script, env))
	}

	// TODO check if a task has been defined multiple times
	return processes, nil
}
