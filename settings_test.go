package main

import "testing"

func TestNormalizeConfigAddsDefaults(t *testing.T) {
	cfg := newDefaultConfig()
	cfg.Position = Position{X: 7, Y: 9}

	NormalizeConfig(cfg)

	if cfg.Position.X != 7 || cfg.Position.Y != 9 {
		t.Fatalf("position changed: %+v", cfg.Position)
	}
	if !cfg.Visible {
		t.Fatal("expected visible default true")
	}
	if cfg.ScalePercent != defaultScalePercent {
		t.Fatalf("scale = %d", cfg.ScalePercent)
	}
	if cfg.StepSize != defaultStepSize {
		t.Fatalf("step = %d", cfg.StepSize)
	}
	if cfg.WalkIntervalMs != defaultWalkIntervalMs {
		t.Fatalf("interval = %d", cfg.WalkIntervalMs)
	}
}

func TestNormalizeConfigRejectsInvalidPresetValues(t *testing.T) {
	cfg := newDefaultConfig()
	cfg.Position = Position{X: 1, Y: 2}
	cfg.ScalePercent = 99
	cfg.StepSize = 999
	cfg.WalkIntervalMs = 42

	NormalizeConfig(cfg)

	if cfg.ScalePercent != defaultScalePercent {
		t.Fatalf("scale = %d", cfg.ScalePercent)
	}
	if cfg.StepSize != defaultStepSize {
		t.Fatalf("step = %d", cfg.StepSize)
	}
	if cfg.WalkIntervalMs != defaultWalkIntervalMs {
		t.Fatalf("interval = %d", cfg.WalkIntervalMs)
	}
}

func TestPetSizeForScale(t *testing.T) {
	w, h := petSizeForScale(120)

	if w != 120 {
		t.Fatalf("width = %d", w)
	}
	if h != 183 {
		t.Fatalf("height = %d", h)
	}
}
