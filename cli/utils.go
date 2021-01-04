package cli

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"os"
	"strings"
)

const (
	ColorBlue   = "\033[1;34m"
	ColorGreen  = "\033[1;36m"
	ColorYellow = "\033[1;33m"
	ColorRed    = "\033[1;31m"
	ColorReset  = "\033[0m"

	ColorSuccess = ColorGreen
	ColorWarning = ColorYellow
	ColorError   = ColorRed
)

func Colorize(color, text string) string {
	return strings.Join([]string{color, text, ColorReset}, "")
}

// UserSuccess prints a colorized success message
func UserSuccess(msg string, format ...interface{}) {
	msg = fmt.Sprintf(msg, format...)
	fmt.Println(Colorize(ColorSuccess, msg))
}

// UserWarning prints a colorized warning message
func UserWarning(msg string, format ...interface{}) {
	msg = fmt.Sprintf("WARNING: "+msg, format...)
	fmt.Println(Colorize(ColorWarning, msg))
}

// UserError prints a colorized error message and terminates with a non-zero exit code
func UserError(msg string, format ...interface{}) {
	msg = fmt.Sprintf("ERROR: "+msg, format...)
	fmt.Println(Colorize(ColorError, msg))
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
