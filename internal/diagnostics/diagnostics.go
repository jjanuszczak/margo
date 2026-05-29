package diagnostics

import (
	"fmt"
	"io"
	"strings"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Diagnostic struct {
	Severity Severity
	Code     string
	Message  string
	Path     string
	Line     int
}

type Report struct {
	Items []Diagnostic
}

func (r *Report) Add(d Diagnostic) {
	r.Items = append(r.Items, d)
}

func (r Report) HasErrors() bool {
	for _, item := range r.Items {
		if item.Severity == SeverityError {
			return true
		}
	}

	return false
}

func (d Diagnostic) String() string {
	var b strings.Builder

	if d.Code != "" {
		b.WriteString("[")
		b.WriteString(d.Code)
		b.WriteString("] ")
	}

	b.WriteString(string(d.Severity))
	b.WriteString(": ")
	b.WriteString(d.Message)

	if d.Path != "" {
		b.WriteString(" (")
		b.WriteString(d.Path)
		if d.Line > 0 {
			b.WriteString(fmt.Sprintf(":%d", d.Line))
		}
		b.WriteString(")")
	}

	return b.String()
}

func WriteReport(w io.Writer, report Report) {
	for _, item := range report.Items {
		fmt.Fprintln(w, item.String())
	}
}
