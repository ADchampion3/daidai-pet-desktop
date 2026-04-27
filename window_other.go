//go:build !windows

package main

import "context"

func setWindowClickThrough(ctx context.Context, enabled bool) error {
	return nil
}

func setWindowManagerExclusion(ctx context.Context) error {
	return nil
}

func setWindowBoundsAbsolute(ctx context.Context, x, y, width, height int) error {
	return nil
}
