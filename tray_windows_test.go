//go:build windows

package main

import (
	"testing"

	"golang.org/x/sys/windows"
)

func TestTrayWindowProcDispatchesWMCommand(t *testing.T) {
	withDisplayLayoutForTest(t, displayLayout{
		{Left: 0, Top: 0, Right: 1920, Bottom: 1080},
		{Left: 1920, Top: 0, Right: 3200, Bottom: 1024},
	})
	app := &App{cfg: newDefaultConfig()}
	controller := &trayController{app: app}
	hwnd := windows.Handle(1)
	controller.hwnd = hwnd
	trayControllers.Store(hwnd, controller)
	defer trayControllers.Delete(hwnd)

	trayWindowProc(uintptr(hwnd), wmCommand, menuCommandShowHide, 0)
	if app.currentVisible() {
		t.Fatal("currentVisible() = true after show/hide toggle")
	}

	trayWindowProc(uintptr(hwnd), wmCommand, menuCommandSizeBase+2, 0)
	if got := app.currentScalePercent(); got != 120 {
		t.Fatalf("currentScalePercent() = %d", got)
	}

	trayWindowProc(uintptr(hwnd), wmCommand, menuCommandStepBase+3, 0)
	if got := app.currentStepSize(); got != 40 {
		t.Fatalf("currentStepSize() = %d", got)
	}

	trayWindowProc(uintptr(hwnd), wmCommand, menuCommandWalkBase+2, 0)
	if got := app.currentWalkInterval(); got.Milliseconds() != 300 {
		t.Fatalf("currentWalkInterval() = %s", got)
	}

	trayWindowProc(uintptr(hwnd), wmCommand, menuCommandToggleDrag, 0)
	if app.currentDragEnabled() {
		t.Fatal("currentDragEnabled() = true after tray toggle")
	}

	trayWindowProc(uintptr(hwnd), wmCommand, menuCommandDisplayBase+1, 0)
	if app.currentDisplayIndex() != 1 {
		t.Fatalf("currentDisplayIndex() = %d", app.currentDisplayIndex())
	}
}

func TestTrayWindowProcHandlesWMCloseWithoutTrayHandle(t *testing.T) {
	controller := &trayController{}
	trayControllers.Store(windows.Handle(0), controller)
	defer trayControllers.Delete(windows.Handle(0))

	if got := trayWindowProc(0, wmClose, 0, 0); got != 0 {
		t.Fatalf("trayWindowProc(...) = %d", got)
	}
}
