package printers

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/fatih/color"

	"github.com/golangci/golangci-lint/pkg/logutils"
	"github.com/golangci/golangci-lint/pkg/result"
)

type Tab struct {
	printLinterName bool
	useColors       bool

	log logutils.Log
	w   io.Writer
}

func NewTab(printLinterName, useColors bool, log logutils.Log, w io.Writer) *Tab {
	return &Tab{
		printLinterName: printLinterName,
		useColors:       useColors,
		log:             log,
		w:               w,
	}
}

func (p *Tab) SprintfColored(ca color.Attribute, format string, args ...any) string {
	c := color.New(ca)

	if !p.useColors {
		c.DisableColor()
	}

	return c.Sprintf(format, args...)
}

func (p *Tab) Print(issues []result.Issue) error {
	w := tabwriter.NewWriter(p.w, 0, 0, 2, ' ', 0)

	for i := range issues {
		p.printIssue(&issues[i], w)
	}

	if err := w.Flush(); err != nil {
		p.log.Warnf("Can't flush tab writer: %s", err)
	}

	return nil
}

func (p *Tab) printIssue(i *result.Issue, w io.Writer) {
	text := p.SprintfColored(color.FgRed, "%s", i.Text)
	if p.printLinterName {
		text = fmt.Sprintf("%s\t%s", i.FromLinter, text)
	}

	pos := p.SprintfColored(color.Bold, "%s:%d", i.FilePath(), i.Line())
	if i.Pos.Column != 0 {
		pos += fmt.Sprintf(":%d", i.Pos.Column)
	}

	fmt.Fprintf(w, "%s\t%s\n", pos, text)
}
