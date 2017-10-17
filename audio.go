package main

/**
 * Audio status
 */

import (
	"log"

	ui "github.com/gizak/termui"
	"github.com/sqp/pulseaudio"
)

////////////////////////////////////////////
// Widget: Audio
////////////////////////////////////////////

type AudioWidget struct {
	widget        *ui.Gauge
	pulse         *pulseaudio.Client
	volumePercent uint32
	isMuted       bool
}

func NewAudioWidget() *AudioWidget {
	// Create base element
	e := ui.NewGauge()
	e.Height = 3
	e.Border = true
	e.BorderLabel = "Audio"

	// Connect to pulseaudio daemon
	pulse, err := pulseaudio.New()
	if err != nil {
		log.Printf("Error connecting to pulse daemon: %v", err)
		pulse = nil
	}

	// Create widget
	w := &AudioWidget{
		widget:        e,
		pulse:         pulse,
		volumePercent: 0,
		isMuted:       false,
	}

	// Register listener
	if pulse != nil {
		pulse.Register(w)
	}

	w.update()
	w.resize()

	return w
}

func (w *AudioWidget) getGridWidget() ui.GridBufferer {
	return w.widget
}

func (w *AudioWidget) update() {
	if w.pulse == nil {
		w.widget.BorderLabel = "Audio"
		w.widget.Percent = 0
		w.widget.Label = "UNSUPPORTED"
		w.widget.LabelAlign = ui.AlignCenter
		w.widget.PercentColor = ui.ColorMagenta + ui.AttrBold
	} else {
		// Just query status
		sink := w.getBestSink()

		if sink != nil {
			// Load information about this sink
			muted, mutedErr := sink.Bool("Mute")

			if mutedErr == nil {
				w.isMuted = muted
			} else {
				w.isMuted = false
			}

			volume, volErr := sink.ListUint32("Volume")

			if volErr == nil {
				// Convert to a percent (with shitty rounding)
				volPercent := (volume[0] * 1000) / 65536
				volPercent = (volPercent + 5) / 10

				w.volumePercent = volPercent
			} else {
				w.volumePercent = 0
			}
		}

		w.widget.Percent = int(w.volumePercent)
		w.widget.Label = "{{percent}}%"
		w.widget.LabelAlign = ui.AlignRight
		w.widget.PercentColor = ui.ColorWhite + ui.AttrBold
		w.widget.PercentColorHighlighted = w.widget.PercentColor

		if w.isMuted {
			w.widget.BarColor = ui.ColorRed
		} else {
			w.widget.BarColor = ui.ColorGreen
		}
	}
}

func (w *AudioWidget) getBestSink() *pulseaudio.Object {
	fallbackSink, fallbackErr := w.pulse.Core().ObjectPath("FallbackSink")

	if fallbackErr == nil {
		return w.pulse.Device(fallbackSink)
	} else {
		sinks, sinkErr := w.pulse.Core().ListPath("Sinks")

		if sinkErr == nil {
			// Take the first one
			return w.pulse.Device(sinks[0])
		}
	}

	return nil
}

func (w *AudioWidget) resize() {
	// Do nothing
}
