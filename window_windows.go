//go:build windows

package main

import (
	"context"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	gwlExStyle      = -20
	wsExTransparent = 0x00000020
	swpNoZOrder     = 0x0004
	swpNoActivate   = 0x0010
	windowTitle     = "Pet"
)

var (
	procFindWindowW       = user32.NewProc("FindWindowW")
	procGetWindowLongPtrW = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW = user32.NewProc("SetWindowLongPtrW")
	procSetWindowPos      = user32.NewProc("SetWindowPos")
)

func setWindowClickThrough(ctx context.Context, enabled bool) error {
	if ctx == nil {
		return nil
	}

	hwnd, err := findPetWindow()
	if err != nil {
		return err
	}

	exStyleIndex := windowLongIndex(gwlExStyle)
	style, _, _ := procGetWindowLongPtrW.Call(hwnd, exStyleIndex)
	nextStyle := style
	if enabled {
		nextStyle |= wsExTransparent
	} else {
		nextStyle &^= wsExTransparent
	}
	if nextStyle == style {
		return nil
	}

	ret, _, callErr := procSetWindowLongPtrW.Call(hwnd, exStyleIndex, nextStyle)
	if ret == 0 && callErr != windows.ERROR_SUCCESS {
		return fmt.Errorf("SetWindowLongPtrW: %w", callErr)
	}
	return nil
}

func windowLongIndex(value int32) uintptr {
	return uintptr(uint32(value))
}

func setWindowBoundsAbsolute(ctx context.Context, x, y, width, height int) error {
	if ctx == nil {
		return nil
	}

	hwnd, err := findPetWindow()
	if err != nil {
		return err
	}

	ret, _, callErr := procSetWindowPos.Call(
		hwnd,
		0,
		uintptr(int32(x)),
		uintptr(int32(y)),
		uintptr(int32(width)),
		uintptr(int32(height)),
		swpNoZOrder|swpNoActivate,
	)
	if ret == 0 && callErr != windows.ERROR_SUCCESS {
		return fmt.Errorf("SetWindowPos: %w", callErr)
	}
	return nil
}

func findPetWindow() (uintptr, error) {
	title, err := windows.UTF16PtrFromString(windowTitle)
	if err != nil {
		return 0, err
	}

	hwnd, _, callErr := procFindWindowW.Call(0, uintptr(unsafe.Pointer(title)))
	if hwnd == 0 {
		return 0, fmt.Errorf("FindWindowW %q: %w", windowTitle, callErr)
	}
	return hwnd, nil
}
