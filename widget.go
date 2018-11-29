package main

/**
 * My widget wraper(s).
 */

import (
	"time"

	ui "github.com/gizak/termui"
)

////////////////////////////////////////////
// Utility: Widgets
////////////////////////////////////////////

type CAHWidget interface {
	getGridWidget() ui.GridBufferer
	update()
	resize()
}

type UpdateInterval interface {
	getUpdateInterval() time.Duration
	getLastUpdated() *time.Time
	setLastUpdated(t time.Time)
}

func shouldUpdate(updater UpdateInterval) bool {
	now := time.Now()
	lastUpdated := updater.getLastUpdated()

	if lastUpdated == nil {
		// First time, execute it
		updater.setLastUpdated(now)
		return true
	} else {
		// Has enough time passed?
		elapsed := now.Sub(*lastUpdated)

		if elapsed.Nanoseconds() > updater.getUpdateInterval().Nanoseconds() {
			updater.setLastUpdated(time.Now())
			return true
		}
	}

	return false
}

type TempWidget struct {
	widget *ui.Paragraph
}

func NewTempWidget(l string) *TempWidget {
	p := ui.NewParagraph(l)
	p.Height = 5
	p.BorderLabel = l

	// Create our widget
	w := &TempWidget{
		widget: p,
	}

	// Invoke its functions
	w.update()
	w.resize()

	return w
}

func (w *TempWidget) getGridWidget() ui.GridBufferer {
	return w.widget
}

func (w *TempWidget) update() {
	// Do nothing
}

func (w *TempWidget) resize() {
	// Do nothing
}
