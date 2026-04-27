package main

import (
	"testing"
	"time"
)

func TestMovementUsesConfigurableStepSize(t *testing.T) {
	m := NewMovement(50, 20, 0, 200, 0, 200, 30, 150*time.Millisecond)
	m.direction = Right
	m.running = true

	m.doMove()

	if m.x != 80 {
		t.Fatalf("x = %d", m.x)
	}
	m.Stop()
}

func TestMovementUpdateSettings(t *testing.T) {
	m := NewMovement(50, 20, 0, 200, 0, 200, 20, 150*time.Millisecond)

	m.UpdateSettings(10, 300*time.Millisecond, 5, 95, 10, 90)

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stepSize != 10 {
		t.Fatalf("step = %d", m.stepSize)
	}
	if m.moveInterval != 300*time.Millisecond {
		t.Fatalf("interval = %s", m.moveInterval)
	}
	if m.minX != 5 || m.maxX != 95 {
		t.Fatalf("x bounds = %d..%d", m.minX, m.maxX)
	}
	if m.minY != 10 || m.maxY != 90 {
		t.Fatalf("y bounds = %d..%d", m.minY, m.maxY)
	}
}

func TestMovementMovesVerticallyWhenBouncingAtEdge(t *testing.T) {
	m := NewMovement(190, 100, 0, 200, 0, 200, 30, 150*time.Millisecond)
	m.direction = Right
	m.running = true

	m.doMove()

	if m.x != 200 {
		t.Fatalf("x = %d", m.x)
	}
	if m.direction != Left {
		t.Fatalf("direction = %v", m.direction)
	}
	delta := absInt(m.y - 100)
	if delta < minVerticalBounceOffset || delta > maxVerticalBounceOffset {
		t.Fatalf("vertical delta = %d, y = %d", delta, m.y)
	}
	m.Stop()
}

func TestMovementKeepsVerticalBounceInsideBounds(t *testing.T) {
	m := NewMovement(190, 1, 0, 200, 0, 5, 30, 150*time.Millisecond)
	m.direction = Right
	m.running = true

	m.doMove()

	if m.y < 0 || m.y > 5 {
		t.Fatalf("y = %d", m.y)
	}
	if m.y == 1 {
		t.Fatalf("expected y to change on bounce")
	}
	m.Stop()
}

func TestMovementUpdateSettingsClampsY(t *testing.T) {
	m := NewMovement(50, 120, 0, 200, 0, 200, 20, 150*time.Millisecond)

	m.UpdateSettings(20, 150*time.Millisecond, 0, 200, 10, 90)

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.y != 90 {
		t.Fatalf("y = %d", m.y)
	}
}

func TestMovementRefreshesBoundsForCurrentYBeforeMoving(t *testing.T) {
	m := NewMovement(1800, 100, 0, 1860, 0, 1000, 100, 150*time.Millisecond)
	m.direction = Right
	m.running = true
	m.SetBoundsProvider(func(x, y int) (int, int, int, int) {
		if y < 300 {
			return 0, 1860, 0, 1000
		}
		return 0, 3140, 0, 1000
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

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
