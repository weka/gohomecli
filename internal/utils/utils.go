package utils

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/hokaccha/go-prettyjson"
	"github.com/olekukonko/tablewriter"
)

const (
	ColorRed           = "\033[0;31m"
	ColorGreen         = "\033[0;32m"
	ColorYellow        = "\033[0;33m"
	ColorBlue          = "\033[0;34m"
	ColorMagenta       = "\033[0;35m"
	ColorCyan          = "\033[0;36m"
	ColorBrightRed     = "\033[1;31m"
	ColorBrightGreen   = "\033[1;32m"
	ColorBrightYellow  = "\033[1;33m"
	ColorBrightBlue    = "\033[1;34m"
	ColorBrightMagenta = "\033[1;35m"
	ColorBrightCyan    = "\033[1;36m"

	ColorDarkGrey = "\033[1;30m"
	ColorWhite    = "\033[1;37m"

	ColorReset = "\033[0m"

	ColorOutput  = ColorWhite
	ColorSuccess = ColorBrightCyan
	ColorWarning = ColorBrightYellow
	ColorError   = ColorBrightRed
)

var ErrValidationFailed = errors.New("validation error")

var IsColorOutputSupported bool

func Colorize(color, text string) string {
	if !IsColorOutputSupported {
		return text
	}
	return strings.Join([]string{color, text, ColorReset}, "")
}

func ColorizeJSON(data []byte) []byte {
	if !IsColorOutputSupported {
		return data
	}
	formatter := prettyjson.NewFormatter()
	formatter.Indent = 4
	formatted, err := formatter.Format(data)
	if err != nil {
		UserWarning("Failed to colorize JSON: %s", err)
		return data
	}
	return formatted
}

// UserOutput prints text to stdout. Use this function to output a command's
// return value.
func UserOutput(msg string, format ...interface{}) {
	msg = fmt.Sprintf(msg, format...)
	fmt.Println(Colorize(ColorOutput, msg))
}

// UserOutputJSON is like UserOutput, but for JSON
func UserOutputJSON(data []byte) {
	UserOutput(string(ColorizeJSON(data)))
}

// UserNote prints a colorized info message to stderr. Use this function to
// output a neutral or positive message to the user.
func UserNote(msg string, format ...interface{}) {
	msg = fmt.Sprintf(msg, format...)
	fmt.Fprintln(os.Stderr, Colorize(ColorSuccess, msg))
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

type TableRenderer struct {
	Headers  []string
	Populate func(table *tablewriter.Table)
}

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

type TableRow struct {
	Cells []string
	index int
}

func NewTableRow(numCells int) *TableRow {
	return &TableRow{Cells: make([]string, numCells)}
}

func (row *TableRow) Append(cells ...string) {
	for _, cellText := range cells {
		row.Cells[row.index] = cellText
		row.index++
	}
}

func UnescapeUnicodeCharactersInJSON(_jsonRaw json.RawMessage) (json.RawMessage, error) {
	str, err := strconv.Unquote(strings.Replace(strconv.Quote(string(_jsonRaw)), `\\u`, `\u`, -1))
	if err != nil {
		return nil, err
	}
	return []byte(str), nil
}

func IsFileExists(name string) bool {
	name, err := filepath.EvalSymlinks(name)
	if err != nil {
		return false
	}

	_, err = os.Stat(name)
	return err == nil
}

type commandOpt func(*WrappedCmd) error

var WithStdoutReader = func(cb func(lines chan []byte)) func(cmd *WrappedCmd) error {
	return func(cmd *WrappedCmd) error {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}

		cmd.wg.Add(1) // we need to wait for stderr to closed before we can wait for the command to finish

		lines := make(chan []byte)

		// run callback
		go func() {
			cb(lines)
			cmd.wg.Done()
		}()

		// run scanner
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				bytes := scanner.Bytes()
				if len(bytes) > 0 {
					lines <- bytes
				}
			}
			close(lines)
			cmd.wg.Done()
		}()
		return nil
	}
}

var WithStderrReader = func(cb func(lines chan []byte)) func(cmd *WrappedCmd) error {
	return func(cmd *WrappedCmd) error {
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return err
		}

		cmd.wg.Add(1) // we need to wait for stderr to closed before we can wait for the command to finish

		lines := make(chan []byte)

		// run callback
		go func() {
			cb(lines)
			cmd.wg.Done()
		}()

		// run scanner
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				bytes := scanner.Bytes()
				if len(bytes) > 0 {
					lines <- bytes
				}
			}
			close(lines)
		}()

		return nil
	}
}

var WithStdin = func(stdin io.Reader) func(cmd *WrappedCmd) error {
	return func(cmd *WrappedCmd) error {
		cmd.Stdin = stdin
		return nil
	}
}

type WrappedCmd struct {
	*exec.Cmd
	wg sync.WaitGroup
}

func (c *WrappedCmd) Wait() error {
	c.wg.Wait()
	return c.Cmd.Wait()
}

func ExecCommand(ctx context.Context, name string, args []string, opts ...commandOpt) (*WrappedCmd, error) {
	cmd := WrappedCmd{
		Cmd: exec.CommandContext(ctx, name, args...),
	}

	for _, opt := range opts {
		if err := opt(&cmd); err != nil {
			return nil, err
		}
	}
	return &cmd, cmd.Start()
}

// IsSetP returns true if pointer is not nil and value is not empty
func IsSetP[T comparable](v *T) bool {
	var empty T
	if v != nil && *v != empty {
		return true
	}

	return false
}

func URLSafe(u *url.URL) *url.URL {
	urlSafe := *u
	urlSafe.User = url.UserPassword(u.User.Username(), "[HIDDEN]")
	return &urlSafe
}

type WriteScanner struct {
	io.Writer
	io.Closer
	ErrCloser interface {
		CloseWithError(err error) error
	}
}

func NewWriteScanner(readers ...func([]byte)) WriteScanner {
	reader, writer := io.Pipe()
	go func() {
		scan := bufio.NewScanner(reader)
		for scan.Scan() {
			for _, cb := range readers {
				cb(scan.Bytes())
			}
		}
	}()

	return WriteScanner{
		Writer:    writer,
		Closer:    writer,
		ErrCloser: writer,
	}
}
