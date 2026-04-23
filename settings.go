package main

const (
	basePetWidth  = 100
	basePetHeight = 153
)

const (
	defaultVisible        = true
	defaultScalePercent   = 100
	defaultStepSize       = 20
	defaultWalkIntervalMs = 150
)

var (
	scalePresets        = []int{80, 100, 120, 150}
	stepSizePresets     = []int{10, 20, 30, 40}
	walkIntervalPresets = []int{100, 150, 300, 600}
)

func NormalizeConfig(cfg *Config) {
	if cfg == nil {
		return
	}

	if !containsPreset(scalePresets, cfg.ScalePercent) {
		cfg.ScalePercent = defaultScalePercent
	}
	if !containsPreset(stepSizePresets, cfg.StepSize) {
		cfg.StepSize = defaultStepSize
	}
	if !containsPreset(walkIntervalPresets, cfg.WalkIntervalMs) {
		cfg.WalkIntervalMs = defaultWalkIntervalMs
	}
}

func newDefaultConfig() *Config {
	return &Config{
		Position:       Position{X: 100, Y: 100},
		Visible:        defaultVisible,
		ScalePercent:   defaultScalePercent,
		StepSize:       defaultStepSize,
		WalkIntervalMs: defaultWalkIntervalMs,
	}
}

func containsPreset(presets []int, value int) bool {
	for _, preset := range presets {
		if preset == value {
			return true
		}
	}
	return false
}

func petSizeForScale(scalePercent int) (int, int) {
	return basePetWidth * scalePercent / 100, basePetHeight * scalePercent / 100
}
