package main

import "math"

type displayRect struct {
	Left   int
	Top    int
	Right  int
	Bottom int
}

type displayLayout []displayRect

type movementBoundsProvider func(x, y int) (int, int)

var displayLayoutProvider = readDisplayLayout

func getDisplayLayout() displayLayout {
	displays := displayLayoutProvider()
	if len(displays) == 0 {
		x, y, w, h := getVirtualScreenBounds()
		displays = displayLayout{{Left: x, Top: y, Right: x + w, Bottom: y + h}}
	}
	return displays
}

func (l displayLayout) bounds() displayRect {
	if len(l) == 0 {
		return displayRect{}
	}
	bounds := l[0]
	for _, display := range l[1:] {
		if display.Left < bounds.Left {
			bounds.Left = display.Left
		}
		if display.Top < bounds.Top {
			bounds.Top = display.Top
		}
		if display.Right > bounds.Right {
			bounds.Right = display.Right
		}
		if display.Bottom > bounds.Bottom {
			bounds.Bottom = display.Bottom
		}
	}
	return bounds
}

func (l displayLayout) clampWindowPosition(x, y, width, height int) (int, int) {
	if len(l) == 0 {
		return x, y
	}
	for _, display := range l {
		if display.containsWindow(x, y, width, height) {
			return x, y
		}
	}

	centerX := x + width/2
	centerY := y + height/2
	nearest := l[0]
	nearestDistance := nearest.distanceSquaredToPoint(centerX, centerY)
	for _, display := range l[1:] {
		if distance := display.distanceSquaredToPoint(centerX, centerY); distance < nearestDistance {
			nearest = display
			nearestDistance = distance
		}
	}
	return nearest.clampWindowPosition(x, y, width, height)
}

func (l displayLayout) clampWindowPositionToDisplay(index int, x, y, width, height int) (int, int) {
	if len(l) == 0 {
		return x, y
	}
	display, ok := l.display(index)
	if !ok {
		display = l[0]
	}
	return display.clampWindowPosition(x, y, width, height)
}

func (l displayLayout) movementBounds(x, y, width, height int) (int, int) {
	if len(l) == 0 {
		return x, x
	}

	var intervals []displayInterval
	for _, display := range l {
		if !display.canFitWindowAtY(y, height) {
			continue
		}
		maxX := display.Right - width
		if maxX < display.Left {
			maxX = display.Left
		}
		intervals = append(intervals, displayInterval{Min: display.Left, Max: maxX})
	}
	if len(intervals) == 0 {
		clampedX, clampedY := l.clampWindowPosition(x, y, width, height)
		return l.movementBounds(clampedX, clampedY, width, height)
	}

	intervals = mergeDisplayIntervals(intervals, width)
	chosen := intervals[0]
	bestDistance := chosen.distanceToX(x)
	for _, interval := range intervals[1:] {
		if distance := interval.distanceToX(x); distance < bestDistance {
			chosen = interval
			bestDistance = distance
		}
	}
	return chosen.Min, chosen.Max
}

func (l displayLayout) movementBoundsForDisplay(index int, x, y, width, height int) (int, int) {
	if len(l) == 0 {
		return x, x
	}
	display, ok := l.display(index)
	if !ok {
		display = l[0]
	}

	maxX := display.Right - width
	if maxX < display.Left {
		maxX = display.Left
	}
	return display.Left, maxX
}

func (l displayLayout) display(index int) (displayRect, bool) {
	if index < 0 || index >= len(l) {
		return displayRect{}, false
	}
	return l[index], true
}

func (r displayRect) containsWindow(x, y, width, height int) bool {
	return x >= r.Left && y >= r.Top && x+width <= r.Right && y+height <= r.Bottom
}

func (r displayRect) centerWindowPosition(width, height int) (int, int) {
	x := r.Left + (r.Right-r.Left-width)/2
	y := r.Top + (r.Bottom-r.Top-height)/2
	return r.clampWindowPosition(x, y, width, height)
}

func (r displayRect) canFitWindowAtY(y, height int) bool {
	return y >= r.Top && y+height <= r.Bottom
}

func (r displayRect) clampWindowPosition(x, y, width, height int) (int, int) {
	maxX := r.Right - width
	maxY := r.Bottom - height
	if maxX < r.Left {
		maxX = r.Left
	}
	if maxY < r.Top {
		maxY = r.Top
	}
	return clampInt(x, r.Left, maxX), clampInt(y, r.Top, maxY)
}

func (r displayRect) distanceSquaredToPoint(x, y int) int {
	dx := 0
	if x < r.Left {
		dx = r.Left - x
	} else if x > r.Right {
		dx = x - r.Right
	}

	dy := 0
	if y < r.Top {
		dy = r.Top - y
	} else if y > r.Bottom {
		dy = y - r.Bottom
	}
	return dx*dx + dy*dy
}

type displayInterval struct {
	Min int
	Max int
}

func (i displayInterval) distanceToX(x int) int {
	if x < i.Min {
		return i.Min - x
	}
	if x > i.Max {
		return x - i.Max
	}
	return 0
}

func mergeDisplayIntervals(intervals []displayInterval, windowWidth int) []displayInterval {
	for i := 1; i < len(intervals); i++ {
		current := intervals[i]
		j := i - 1
		for ; j >= 0 && intervals[j].Min > current.Min; j-- {
			intervals[j+1] = intervals[j]
		}
		intervals[j+1] = current
	}

	merged := intervals[:1]
	for _, interval := range intervals[1:] {
		last := &merged[len(merged)-1]
		if interval.Min <= last.Max+windowWidth {
			if interval.Max > last.Max {
				last.Max = interval.Max
			}
			continue
		}
		merged = append(merged, interval)
	}
	return merged
}

func clampInt(value, minValue, maxValue int) int {
	return int(math.Min(float64(maxValue), math.Max(float64(minValue), float64(value))))
}
