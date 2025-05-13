package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hokaccha/go-prettyjson"
)

const (
	colorRed           = "\033[0;31m"
	colorGreen         = "\033[0;32m"
	colorYellow        = "\033[0;33m"
	colorBlue          = "\033[0;34m"
	colorMagenta       = "\033[0;35m"
	colorCyan          = "\033[0;36m"
	colorBrightRed     = "\033[1;31m"
	colorBrightGreen   = "\033[1;32m"
	colorBrightYellow  = "\033[1;33m"
	colorBrightBlue    = "\033[1;34m"
	colorBrightMagenta = "\033[1;35m"
	colorBrightCyan    = "\033[1;36m"

	colorDarkGrey = "\033[1;30m"
	colorWhite    = "\033[1;37m"

	colorReset = "\033[0m"

	colorOutput  = colorWhite
	colorSuccess = colorBrightCyan
	colorWarning = colorBrightYellow
	colorError   = colorBrightRed

	programErrorExitCode = 2
)

var (
	// ErrValidationFailed is an error that occurs when validation fails
	ErrValidationFailed = errors.New("validation error")

	// ErrColorOutput is an error that occurs when color output is not supported
	ErrColorOutput = errors.New("color output is not supported")

	// IsColorOutputSupported returns true if color output is supported
	//nolint:gochecknoglobals // this is a global variable
	IsColorOutputSupported = false
)

// Colorize colorizes a text string
func Colorize(color, text string) string {
	if !IsColorOutputSupported {
		return text
	}

	return strings.Join([]string{color, text, colorReset}, "")
}

// ColorizeJSON colorizes a JSON string
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
func UserOutput(msg string, format ...any) {
	msg = fmt.Sprintf(msg, format...)
	fmt.Println(Colorize(colorOutput, msg))
}

// UserOutputJSON is like UserOutput, but for JSON
func UserOutputJSON(data []byte) {
	UserOutput(string(ColorizeJSON(data)))
}

// UserNote prints a colorized info message to stderr. Use this function to
// output a neutral or positive message to the user.
func UserNote(msg string, format ...any) {
	msg = fmt.Sprintf(msg, format...)
	fmt.Fprintln(os.Stderr, Colorize(colorSuccess, msg))
}

// UserWarning prints a colorized warning message to stderr
func UserWarning(msg string, format ...any) {
	msg = fmt.Sprintf("WARNING: "+msg, format...)
	fmt.Fprintln(os.Stderr, Colorize(colorWarning, msg))
}

// UserError prints a colorized error message to stderr, and terminates with a
// non-zero exit code
func UserError(msg string, format ...any) {
	msg = fmt.Sprintf("ERROR: "+msg, format...)
	fmt.Fprintln(os.Stderr, Colorize(colorError, msg))
	os.Exit(programErrorExitCode)
}

// UnescapeUnicodeCharactersInJSON unescapes unicode characters in a JSON string
func UnescapeUnicodeCharactersInJSON(_jsonRaw json.RawMessage) (json.RawMessage, error) {
	str, err := strconv.Unquote(strings.ReplaceAll(strconv.Quote(string(_jsonRaw)), `\\u`, `\u`))
	if err != nil {
		return nil, err
	}

	return []byte(str), nil
}

// IsFileExists returns true if the file exists
func IsFileExists(name string) bool {
	name, err := filepath.EvalSymlinks(name)
	if err != nil {
		return false
	}

	_, err = os.Stat(name)

	return err == nil
}

// IsSetP returns true if pointer is not nil and value is not empty
func IsSetP[T comparable](v *T) bool {
	var empty T
	if v != nil && *v != empty {
		return true
	}

	return false
}

// URLSafe returns a new URL with the user password hidden
func URLSafe(u *url.URL) *url.URL {
	urlSafe := *u
	urlSafe.User = url.UserPassword(u.User.Username(), "[HIDDEN]")

	return &urlSafe
}

// GetURLStatusCode returns the status code of a given URL
func GetURLStatusCode(ctx context.Context, statusURL string) (int, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		statusURL,
		nil,
	)
	if err != nil {
		return 0, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			UserWarning("Failed to close response body: %s", err)
		}
	}()

	return resp.StatusCode, nil
}
