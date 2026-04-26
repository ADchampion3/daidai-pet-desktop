package main

import (
	"testing"
	"time"
)

func TestMovementUsesConfigurableStepSize(t *testing.T) {
	m := NewMovement(50, 20, 0, 200, 30, 150*time.Millisecond)
	m.direction = Right
	m.running = true

	m.doMove()

	if m.x != 80 {
		t.Fatalf("x = %d", m.x)
	}
	m.Stop()
}

func TestMovementUpdateSettings(t *testing.T) {
	m := NewMovement(50, 20, 0, 200, 20, 150*time.Millisecond)

	m.UpdateSettings(10, 300*time.Millisecond, 5, 95)

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stepSize != 10 {
		t.Fatalf("step = %d", m.stepSize)
	}
	if m.moveInterval != 300*time.Millisecond {
		t.Fatalf("interval = %s", m.moveInterval)
	}
	if m.minX != 5 || m.maxX != 95 {
		t.Fatalf("bounds = %d..%d", m.minX, m.maxX)
	}
}
