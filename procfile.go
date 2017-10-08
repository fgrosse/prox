package prox

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func ParseProcFile(reader io.Reader, env Environment) ([]Process, error) {
	s := bufio.NewScanner(reader)
	var processes []Process
	var i int
	for s.Scan() {
		line, i := strings.TrimSpace(s.Text()), i+1
		if line == "" || line[0] == '#' {
			continue
		}

		lineParts := strings.SplitN(line, ":", 2)
		if len(lineParts) < 2 {
			return processes, fmt.Errorf("invalid Procfile format at line %d: %s", i, line)
		}

		processes = append(processes, Process{
			Name:   strings.TrimSpace(lineParts[0]),
			Script: strings.TrimSpace(lineParts[1]),
			Env:    env,
		})
	}

	// TODO check if a task has been defined multiple times
	return processes, s.Err()
}
