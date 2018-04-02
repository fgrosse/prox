package prox

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
)

// New creates a github.com/hashicorp/go-multierror.Error which is initialized
// with Format as the ErrorFormat.
func newMultiError() *multierror.Error {
	return &multierror.Error{ErrorFormat: func(es []error) string {
		if len(es) == 0 {
			return ""
		}

		if len(es) == 1 {
			return es[0].Error()
		}

		points := make([]string, len(es))
		for i, err := range es {
			points[i] = fmt.Sprintf("\t* %s", err)
		}

		return "\n" + strings.Join(points, "\n")
	}}
}
