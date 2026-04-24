package main

import "testing"

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
