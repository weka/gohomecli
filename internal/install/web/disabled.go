//go:build !web

package web

import (
	"context"
	"fmt"
)

func IsEnabled() bool {
	return false
}

func ServeConfigurer(ctx context.Context, addr string) error {
	return fmt.Errorf("web component is not included in build")
}
