//go:build !web

package web

import (
	"context"
	"errors"
)

func IsEnabled() bool {
	return false
}

func ServeConfigurer(ctx context.Context, addr string) error {
	return errors.New("web component is not included in build")
}
