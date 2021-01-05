package utils

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"os"
	"strings"
)

const (
	ColorBlue   = "\033[1;34m"
	ColorCyan   = "\033[1;36m"
	ColorYellow = "\033[1;33m"
	ColorRed    = "\033[1;31m"
	ColorReset  = "\033[0m"

	ColorSuccess = ColorCyan
	ColorWarning = ColorYellow
	ColorError   = ColorRed
)

func Colorize(color, text string) string {
	return strings.Join([]string{color, text, ColorReset}, "")
}

// UserOutput prints text to stdout. Use this function to output a command's
// return value.
func UserOutput(msg string, format ...interface{}) {
	msg = fmt.Sprintf(msg, format...)
	fmt.Println(msg)
}

// UserNote prints a colorized info message to stderr. Use this function to
// output a neutral or positive message to the user.
func UserNote(msg string, format ...interface{}) {
	msg = fmt.Sprintf(msg, format...)
	fmt.Println(Colorize(ColorSuccess, msg))
}

// UserWarning prints a colorized warning message to stderr
func UserWarning(msg string, format ...interface{}) {
	msg = fmt.Sprintf("WARNING: "+msg, format...)
	fmt.Fprintln(os.Stderr, Colorize(ColorWarning, msg))
}

// UserError prints a colorized error message to stderr, and terminates with a
// non-zero exit code
func UserError(msg string, format ...interface{}) {
	msg = fmt.Sprintf("ERROR: "+msg, format...)
	fmt.Fprintln(os.Stderr, Colorize(ColorError, msg))
	os.Exit(2)
}

func BoolToYesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

type TableRenderer struct {
	Headers  []string
	Populate func(table *tablewriter.Table)
}

func (tr *TableRenderer) Render() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorder(false)
	table.SetHeader(tr.Headers)
	tr.Populate(table)
	table.Render()
}

func RenderTable(headers []string, populate func(table *tablewriter.Table)) {
	table := &TableRenderer{headers, populate}
	table.Render()
}

func RenderTableRows(headers []string, nextRow func() []string) {
	table := &TableRenderer{
		Headers: headers,
		Populate: func(table *tablewriter.Table) {
			for {
				rowData := nextRow()
				if rowData == nil {
					break
				}
				table.Append(rowData)
			}
		},
	}
	table.Render()
}
