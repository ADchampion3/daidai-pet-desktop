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

func TestMovementRefreshesBoundsForCurrentYBeforeMoving(t *testing.T) {
	m := NewMovement(1800, 100, 0, 1860, 100, 150*time.Millisecond)
	m.direction = Right
	m.running = true
	m.SetBoundsProvider(func(x, y int) (int, int) {
		if y < 300 {
			return 0, 1860
		}
		return 0, 3140
	})

	m.doMove()

	if m.x != 1860 {
		t.Fatalf("x = %d", m.x)
	}
	if m.direction != Left {
		t.Fatalf("direction = %v", m.direction)
	}
	m.Stop()
}
