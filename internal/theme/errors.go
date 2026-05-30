package theme

import (
	"errors"
	"fmt"
)

type Error struct {
	Path    string
	Line    int
	Message string
}

func (e *Error) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d: %s", e.Path, e.Line, e.Message)
	}
	if e.Path != "" {
		return fmt.Sprintf("%s: %s", e.Path, e.Message)
	}
	return e.Message
}

func AsError(err error) (*Error, bool) {
	var themeErr *Error
	if errors.As(err, &themeErr) {
		return themeErr, true
	}
	return nil, false
}
