//go:build !windows

package main

import "context"

func setWindowClickThrough(ctx context.Context, enabled bool) error {
	return nil
}
