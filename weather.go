package main

/**
 * Weather goodies.
 */

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	ui "github.com/gizak/termui"
)

////////////////////////////////////////////
// Widget: Twitter
////////////////////////////////////////////

const WeatherWidgetUpdateInterval = 1 * time.Hour

type WeatherWidget struct {
	location    string
	widget      *ui.Par
	lastUpdated *time.Time
}

func NewWeatherWidget(location string) *WeatherWidget {
	// Create base element
	e := ui.NewPar("")
	e.Border = true
	e.Height = 7
	e.BorderLabelFg = ui.ColorGreen

	// Create widget
	w := &WeatherWidget{
		location: location,
		widget:   e,
	}

	w.update()
	w.resize()

	return w
}

func (w *WeatherWidget) getGridWidget() ui.GridBufferer {
	return w.widget
}

func (w *WeatherWidget) update() {
	if shouldUpdate(w) {
		// Load weather info

		client := &http.Client{}

		req, err := http.NewRequest("GET", fmt.Sprintf("http://wttr.in/%s?T0q", w.location), nil)

		if err != nil {
			log.Printf("Error creating request: %v", err)
		} else {
			req.Header.Set("User-Agent", "curl")

			resp, err := client.Do(req)
			if err != nil {
				// handle err
				log.Printf("Error loading weather: %v", err)
			} else {
				defer resp.Body.Close()
				body, err := ioutil.ReadAll(resp.Body)

				if err != nil {
					log.Printf("Failed to read body from weather: %v", err)
				} else {
					parts := strings.SplitN(string(body), "\n", 3)

					// Header
					w.widget.BorderLabel = parts[0]

					// Weather
					// TODO: Figure out how to ansi up the weather somehow...  Or preserve the ANSI from the service (T)
					w.widget.Text = stripANSI(parts[2])
					w.widget.Text = strings.TrimRight(w.widget.Text, " \t\n")
				}
			}
		}
	}
}

func (w *WeatherWidget) resize() {
	// Do nothing
}

func (w *WeatherWidget) getUpdateInterval() time.Duration {
	return WeatherWidgetUpdateInterval
}

func (w *WeatherWidget) getLastUpdated() *time.Time {
	return w.lastUpdated
}

func (w *WeatherWidget) setLastUpdated(t time.Time) {
	w.lastUpdated = &t
}
