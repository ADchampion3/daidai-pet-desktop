//go:build !windows

package main

type trayMenu struct{}

func newTrayMenu(app *App) (*trayMenu, error) {
	return &trayMenu{}, nil
}

func (t *trayMenu) Dispose() {}

func trayDragLabel(enabled bool) string {
	if enabled {
		return "Disable Drag"
	}
	return "Enable Drag"
}
