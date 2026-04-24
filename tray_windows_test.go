//go:build windows

package main

import "testing"

func TestTrayHandleCommandUpdatesConfig(t *testing.T) {
	app := &App{cfg: newDefaultConfig()}
	controller := &trayController{app: app}

	controller.handleCommand(menuCommandShowHide)
	if app.currentVisible() {
		t.Fatal("currentVisible() = true after show/hide toggle")
	}

	controller.handleCommand(menuCommandSizeBase + 2)
	if got := app.currentScalePercent(); got != 120 {
		t.Fatalf("currentScalePercent() = %d", got)
	}

	controller.handleCommand(menuCommandStepBase + 3)
	if got := app.currentStepSize(); got != 40 {
		t.Fatalf("currentStepSize() = %d", got)
	}

	controller.handleCommand(menuCommandWalkBase + 2)
	if got := app.currentWalkInterval(); got.Milliseconds() != 300 {
		t.Fatalf("currentWalkInterval() = %s", got)
	}
}
