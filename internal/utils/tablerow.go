package utils

// TableRow is a row in a table
type TableRow struct {
	Cells []string
	index int
}

// NewTableRow creates a new TableRow with the given number of cells
func NewTableRow(numCells int) *TableRow {
	return &TableRow{Cells: make([]string, numCells)}
}

// Append appends a cell to the row
func (row *TableRow) Append(cells ...string) {
	for _, cellText := range cells {
		row.Cells[row.index] = cellText
		row.index++
	}
}
