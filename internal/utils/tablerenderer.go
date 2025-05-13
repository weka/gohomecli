package utils

import (
	"os"

	"github.com/olekukonko/tablewriter"
)

// TableRenderer is a helper struct for rendering tables
type TableRenderer struct {
	Populate func(table *tablewriter.Table)
	Headers  []string
}

// Render renders a table
func (tr *TableRenderer) Render() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorder(false)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetHeaderLine(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetTablePadding("aaa")
	table.SetHeader(tr.Headers)
	table.SetAutoFormatHeaders(false)
	table.SetHeaderColor(tr.getHeaderColors()...)
	tr.Populate(table)
	table.Render()
}

func (tr *TableRenderer) getHeaderColors() []tablewriter.Colors {
	result := make([]tablewriter.Colors, len(tr.Headers))
	for i := range tr.Headers {
		result[i] = tablewriter.Colors{tablewriter.FgBlueColor}
	}

	return result
}

// RenderTable renders a table with the given headers and a function to populate the rows
func RenderTable(headers []string, populate func(table *tablewriter.Table)) {
	table := &TableRenderer{
		Headers:  headers,
		Populate: populate,
	}
	table.Render()
}

// RenderTableRows renders a table with the given headers and a function to populate the rows
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
