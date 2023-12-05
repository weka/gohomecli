//go:build !web

package web

import "fmt"

func IsEnabled() bool {
	return false
}

func ListenAndServe(addr string) error {
	return fmt.Errorf("web component is not enabled")
}
