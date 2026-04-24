//go:build !windows

package main

type trayController struct {
	refreshFunc func()
	destroyFunc func()
}

func newTrayController(app *App) (*trayController, error) {
	return &trayController{}, nil
}

func (t *trayController) Refresh() {
	if t != nil && t.refreshFunc != nil {
		t.refreshFunc()
	}
}

func (t *trayController) Destroy() {
	if t != nil && t.destroyFunc != nil {
		t.destroyFunc()
	}
}
