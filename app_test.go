package main

import "testing"

func TestNewDefaultConfigEnablesDrag(t *testing.T) {
	cfg := newDefaultConfig()
	if !cfg.DragEnabled {
		t.Fatal("expected drag to be enabled by default")
	}
}

func TestSetDragEnabledUpdatesConfig(t *testing.T) {
	app := NewApp()
	app.cfg = newDefaultConfig()

	app.SetDragEnabled(false)
	if app.dragEnabled() {
		t.Fatal("expected drag to be disabled")
	}

	app.SetDragEnabled(true)
	if !app.dragEnabled() {
		t.Fatal("expected drag to be enabled")
	}
}

func TestSetDragStartIgnoredWhenDragDisabled(t *testing.T) {
	app := NewApp()
	app.cfg = newDefaultConfig()
	app.SetDragEnabled(false)

	app.SetDragStart()

	if app.isDragActive() {
		t.Fatal("expected drag start to be ignored when drag is disabled")
	}
}

func TestDisablingDragStopsActiveDrag(t *testing.T) {
	app := NewApp()
	app.cfg = newDefaultConfig()
	app.movement = NewMovement(100, 100, 0, 300)
	app.mu.Lock()
	app.isDragging = true
	app.dragSeq = 7
	app.dragStop = make(chan struct{})
	app.mu.Unlock()

	app.SetDragEnabled(false)

	if app.isDragActive() {
		t.Fatal("expected active drag to stop when drag is disabled")
	}

	app.movement.mu.Lock()
	running := app.movement.running
	app.movement.mu.Unlock()
	app.movement.Stop()
	if !running {
		t.Fatal("expected movement to resume after drag is stopped")
	}
}

func TestToggleDragEnabledFlipsState(t *testing.T) {
	app := NewApp()
	app.cfg = newDefaultConfig()

	app.ToggleDragEnabled()
	if app.dragEnabled() {
		t.Fatal("expected drag to be disabled after first toggle")
	}

	app.ToggleDragEnabled()
	if !app.dragEnabled() {
		t.Fatal("expected drag to be enabled after second toggle")
	}
}

func TestTrayDragLabelReflectsState(t *testing.T) {
	if got := trayDragLabel(true); got != "Disable Drag" {
		t.Fatalf("enabled label = %q, want %q", got, "Disable Drag")
	}
	if got := trayDragLabel(false); got != "Enable Drag" {
		t.Fatalf("disabled label = %q, want %q", got, "Enable Drag")
	}
}
