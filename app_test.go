package main

import (
	"testing"
	"time"
)

func TestCurrentScalePercentUsesConfiguredValue(t *testing.T) {
	app := &App{cfg: newDefaultConfig()}
	app.cfg.ScalePercent = 120

	if got := app.currentScalePercent(); got != 120 {
		t.Fatalf("currentScalePercent() = %d", got)
	}
}

func TestCurrentScalePercentUsesDefaultWhenConfigMissing(t *testing.T) {
	app := &App{}

	if got := app.currentScalePercent(); got != defaultScalePercent {
		t.Fatalf("currentScalePercent() = %d", got)
	}
}

func TestCurrentWalkIntervalUsesConfiguredMilliseconds(t *testing.T) {
	app := &App{cfg: newDefaultConfig()}
	app.cfg.WalkIntervalMs = 300

	if got := app.currentWalkInterval(); got != 300*time.Millisecond {
		t.Fatalf("currentWalkInterval() = %s", got)
	}
}

func TestMovementBoundsUseScaledWidth(t *testing.T) {
	app := &App{cfg: newDefaultConfig()}
	app.cfg.ScalePercent = 150

	minX, maxX := app.movementBounds()
	screenX, _, screenW, _ := getVirtualScreenBounds()
	expectedWidth, _ := petSizeForScale(150)

	if minX != screenX {
		t.Fatalf("minX = %d", minX)
	}
	if maxX != screenX+screenW-expectedWidth {
		t.Fatalf("maxX = %d", maxX)
	}
}

func TestShowPetBeforeInitialMovementStartSchedulesDelayedStart(t *testing.T) {
	app := &App{
		cfg:      newDefaultConfig(),
		movement: NewMovement(100, 100, 0, 600, defaultStepSize, time.Duration(defaultWalkIntervalMs)*time.Millisecond),
	}

	app.showPet()

	app.mu.RLock()
	timer := app.startupDelayTimer
	started := app.startupMovementStarted
	app.mu.RUnlock()

	if started {
		t.Fatal("startupMovementStarted = true")
	}
	if timer == nil {
		t.Fatal("startupDelayTimer = nil")
	}
	if !timer.Stop() {
		t.Fatal("startupDelayTimer could not be stopped")
	}

	app.movement.mu.Lock()
	running := app.movement.running
	moveTimer := app.movement.moveTimer
	app.movement.mu.Unlock()

	if running {
		t.Fatal("movement.running = true")
	}
	if moveTimer != nil {
		t.Fatal("movement.moveTimer should be nil before startup delay fires")
	}
}

func TestSetVisibleFalseStopsMovementWithoutContext(t *testing.T) {
	app := &App{
		cfg:      newDefaultConfig(),
		movement: NewMovement(100, 100, 0, 600, defaultStepSize, time.Duration(defaultWalkIntervalMs)*time.Millisecond),
	}
	app.movement.Resume()

	app.SetVisible(false)

	if app.currentVisible() {
		t.Fatal("currentVisible() = true")
	}

	app.movement.mu.Lock()
	running := app.movement.running
	moveTimer := app.movement.moveTimer
	app.movement.mu.Unlock()

	if running {
		t.Fatal("movement.running = true")
	}
	if moveTimer != nil {
		t.Fatal("movement.moveTimer should be nil after hide")
	}
}

func TestUpdateMovementSettingsUsesCurrentConfig(t *testing.T) {
	app := &App{
		cfg:      newDefaultConfig(),
		movement: NewMovement(10000, 100, 0, 50, defaultStepSize, time.Duration(defaultWalkIntervalMs)*time.Millisecond),
	}
	app.cfg.ScalePercent = 150
	app.cfg.StepSize = 30
	app.cfg.WalkIntervalMs = 600

	app.updateMovementSettings()

	expectedMinX, expectedMaxX := app.movementBounds()

	app.movement.mu.Lock()
	defer app.movement.mu.Unlock()

	if app.movement.stepSize != 30 {
		t.Fatalf("stepSize = %d", app.movement.stepSize)
	}
	if app.movement.moveInterval != 600*time.Millisecond {
		t.Fatalf("moveInterval = %s", app.movement.moveInterval)
	}
	if app.movement.minX != expectedMinX {
		t.Fatalf("minX = %d", app.movement.minX)
	}
	if app.movement.maxX != expectedMaxX {
		t.Fatalf("maxX = %d", app.movement.maxX)
	}
	if app.movement.x != expectedMaxX {
		t.Fatalf("x = %d", app.movement.x)
	}
}

func TestGetPetSettingsUsesConfiguredScale(t *testing.T) {
	app := &App{cfg: newDefaultConfig()}
	app.cfg.ScalePercent = 150

	settings := app.GetPetSettings()

	if settings["width"] != 150 {
		t.Fatalf("width = %d", settings["width"])
	}
	if settings["height"] != 229 {
		t.Fatalf("height = %d", settings["height"])
	}
	if settings["scalePercent"] != 150 {
		t.Fatalf("scalePercent = %d", settings["scalePercent"])
	}
}
