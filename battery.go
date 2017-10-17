package main

/**
 * Laptop Battery Info
 */

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	ui "github.com/gizak/termui"
)

////////////////////////////////////////////
// Widget: Battery
////////////////////////////////////////////

const BatteryUpdateIntervalSeconds = 10

type BatteryWidget struct {
	widget      *ui.Gauge
	lastUpdated *time.Time
}

func NewBatteryWidget() *BatteryWidget {
	// Create base element
	e := ui.NewGauge()
	e.Height = 3
	e.Border = true

	// Create widget
	w := &BatteryWidget{
		widget:      e,
		lastUpdated: nil,
	}

	w.update()
	w.resize()

	return w
}

func (w *BatteryWidget) getGridWidget() ui.GridBufferer {
	return w.widget
}

func (w *BatteryWidget) update() {
	if shouldUpdate(w) {
		// Load battery info
		output, _, err := execAndGetOutput("ibam-battery-prompt", nil, "-p")

		if err == nil {
			// Parse the output
			lines := strings.Split(output, "\n")
			if len(lines) >= 4 {
				// we have enough
				timeLeft := stripANSI(lines[1])
				isCharging, chargeErr := strconv.ParseBool(lines[2])
				batteryPercent, percentErr := strconv.Atoi(lines[4])

				if chargeErr != nil {
					isCharging = false
					log.Printf("Error reading charge status: '%v' -- %v", lines[2], chargeErr)
				}

				if percentErr != nil {
					batteryPercent = 0
					log.Printf("Error reading battery percent: '%v' -- %v", lines[4], chargeErr)
				}

				battColor := percentToAttribute(batteryPercent, 0, 100, false)

				if isCharging {
					w.widget.BorderLabel = "Battery (charging)"
					w.widget.BorderLabelFg = ui.ColorCyan + ui.AttrBold
				} else {
					w.widget.BorderLabel = "Battery"
					w.widget.BorderLabelFg = battColor
				}

				w.widget.Percent = batteryPercent
				w.widget.BarColor = battColor
				w.widget.Label = fmt.Sprintf("%d%% (%s)", batteryPercent, timeLeft)
				w.widget.LabelAlign = ui.AlignRight
				w.widget.PercentColor = ui.ColorWhite + ui.AttrBold
				//w.widget.PercentColorHighlighted = ui.ColorBlack
				w.widget.PercentColorHighlighted = w.widget.PercentColor
			} else {
				log.Printf("Not enough lines from battery command!  Output: %v", output)
			}
		} else {
			log.Printf("Error executing battery command: %v", err)
		}
	}
}

func (w *BatteryWidget) resize() {
	// Do nothing
}

func (w *BatteryWidget) getUpdateInterval() time.Duration {
	// Update every 10 seconds
	return time.Second * 10
}

func (w *BatteryWidget) getLastUpdated() *time.Time {
	return w.lastUpdated
}

func (w *BatteryWidget) setLastUpdated(t time.Time) {
	w.lastUpdated = &t
}
