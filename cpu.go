package main

/**
 * CPU Information
 */

import (
	"fmt"
	"time"

	linuxproc "github.com/c9s/goprocinfo/linux"
	ui "github.com/gizak/termui"
)

////////////////////////////////////////////
// Widget: CPU
////////////////////////////////////////////

type CPUWidget struct {
	widget *ui.LineChart

	numProcessors      int
	lastStat           linuxproc.CPUStat
	curStat            linuxproc.CPUStat
	cpuPercent         float64
	loadLast1Min       []float64
	loadLast5Min       []float64
	timestamps         []string
	mostRecent1MinLoad float64
	mostRecent5MinLoad float64
}

func NewCPUWidget() *CPUWidget {
	// Create base element
	e := ui.NewLineChart()
	e.Height = 20
	e.Border = true
	e.PaddingTop = 1
	e.LineColor = ui.ColorBlue | ui.AttrBold
	e.AxesColor = ui.ColorYellow

	// Create widget
	w := &CPUWidget{
		widget:       e,
		cpuPercent:   0,
		loadLast1Min: make([]float64, 0),
		loadLast5Min: make([]float64, 0),
		timestamps:   make([]string, 0),
	}

	w.update()
	w.resize()

	return w
}

func (w *CPUWidget) getGridWidget() ui.GridBufferer {
	return w.widget
}

func (w *CPUWidget) update() {
	w.loadProcessorStats()

	loadPercent := float64(w.mostRecent5MinLoad) / float64(w.numProcessors)

	cpuColorString := percentToAttributeString(int(100.0*w.cpuPercent), 0, 100, true)

	loadColor := percentToAttribute(int(100.0*loadPercent), 0, 100, true)
	loadColorString := percentToAttributeString(int(100.0*loadPercent), 0, 100, true)

	w.widget.BorderLabel = fmt.Sprintf("[CPU: %0.2f%%](%s)[───](fg-white)[5m Load: %0.2f](%s)", w.cpuPercent*100, cpuColorString, w.mostRecent5MinLoad, loadColorString)
	w.widget.Data = w.loadLast1Min
	w.widget.DataLabels = w.timestamps

	// Adjust graph axes color by Load value (never bold)
	w.widget.AxesColor = loadColor

}

func (w *CPUWidget) resize() {
	// Update
}

func (w *CPUWidget) loadProcessorStats() {
	// Read /proc/stat for the overall CPU percentage
	stats, statErr := linuxproc.ReadStat("/proc/stat")

	if statErr == nil {
		// Save last two stats records
		w.lastStat = w.curStat
		w.curStat = stats.CPUStatAll
		w.numProcessors = len(stats.CPUStats)

		// Calculate usage percentage
		// from: https://stackoverflow.com/a/23376195

		prevIdle := w.lastStat.Idle + w.lastStat.IOWait
		curIdle := w.curStat.Idle + w.curStat.IOWait

		prevNonIdle := w.lastStat.User + w.lastStat.Nice + w.lastStat.System + w.lastStat.IRQ + w.lastStat.SoftIRQ + w.lastStat.Steal
		curNonIdle := w.curStat.User + w.curStat.Nice + w.curStat.System + w.curStat.IRQ + w.curStat.SoftIRQ + w.curStat.Steal

		prevTotal := prevIdle + prevNonIdle
		curTotal := curIdle + curNonIdle

		//  differentiate: actual value minus the previous one
		totald := curTotal - prevTotal
		idled := curIdle - prevIdle

		w.cpuPercent = (float64(totald - idled)) / float64(totald)
	}

	// Read load average
	loadavg, loadErr := linuxproc.ReadLoadAvg("/proc/loadavg")

	if loadErr == nil {
		w.mostRecent1MinLoad = loadavg.Last1Min
		w.mostRecent5MinLoad = loadavg.Last5Min
		now := time.Now()
		ts := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())

		// Record, keep a fixed number around
		if len(w.loadLast1Min) > (w.widget.Width * 2) {
			w.loadLast1Min = append(w.loadLast1Min[1:], loadavg.Last1Min)
		} else {
			w.loadLast1Min = append(w.loadLast1Min, loadavg.Last1Min)
		}

		if len(w.loadLast5Min) > (w.widget.Width * 2) {
			w.loadLast5Min = append(w.loadLast5Min[1:], loadavg.Last5Min)
		} else {
			w.loadLast5Min = append(w.loadLast5Min, loadavg.Last5Min)
		}

		if len(w.timestamps) > (w.widget.Width * 2) {
			w.timestamps = append(w.timestamps[1:], ts)
		} else {
			w.timestamps = append(w.timestamps, ts)
		}
	}
}
