//go:build windows

package main

import "testing"

func TestWindowManagerExclusionStyleAddsToolAndNoActivate(t *testing.T) {
	style := windowManagerExclusionStyle(0)

	if style&wsExToolWindow == 0 {
		t.Fatal("WS_EX_TOOLWINDOW not set")
	}
	if style&wsExNoActivate == 0 {
		t.Fatal("WS_EX_NOACTIVATE not set")
	}
}

func TestWindowManagerExclusionStylePreservesExistingFlags(t *testing.T) {
	style := windowManagerExclusionStyle(wsExTransparent)

	if style&wsExTransparent == 0 {
		t.Fatal("existing WS_EX_TRANSPARENT flag was cleared")
	}
}

func TestWindowStyleRefreshFlagsDoNotMoveOrResizeWindow(t *testing.T) {
	flags := windowStyleRefreshFlags()

	if flags&swpNoMove == 0 {
		t.Fatal("SWP_NOMOVE not set")
	}
	if flags&swpNoSize == 0 {
		t.Fatal("SWP_NOSIZE not set")
	}
	if flags&swpFrameChanged == 0 {
		t.Fatal("SWP_FRAMECHANGED not set")
	}
}
