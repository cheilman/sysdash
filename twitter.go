package main

/**
 * Load recent tweets from an account.
 */

import (
	"fmt"
	"time"

	ui "github.com/gizak/termui"
)

////////////////////////////////////////////
// Util: Twitter
////////////////////////////////////////////

////////////////////////////////////////////
// Widget: Twitter
////////////////////////////////////////////

const TwitterWidgetUpdateInterval = 10 * time.Minute

type TwitterWidget struct {
	account     string
	widget      *ui.Par
	lastUpdated *time.Time
}

func NewTwitterWidget(account string) *TwitterWidget {
	// Create base element
	e := ui.NewPar("")
	e.Border = true
	e.BorderLabel = fmt.Sprintf("@%s", account)

	// Create widget
	w := &TwitterWidget{
		account: account,
		widget:  e,
	}

	w.update()
	w.resize()

	return w
}

func (w *TwitterWidget) getGridWidget() ui.GridBufferer {
	return w.widget
}

func (w *TwitterWidget) update() {
	if shouldUpdate(w) {
		w.widget.Text = w.account
	}
}

func (w *TwitterWidget) resize() {
	// Do nothing
}

func (w *TwitterWidget) getUpdateInterval() time.Duration {
	return TwitterWidgetUpdateInterval
}

func (w *TwitterWidget) getLastUpdated() *time.Time {
	return w.lastUpdated
}

func (w *TwitterWidget) setLastUpdated(t time.Time) {
	w.lastUpdated = &t
}
