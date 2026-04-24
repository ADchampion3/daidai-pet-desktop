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
