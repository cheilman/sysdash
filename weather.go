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

	ui "github.com/ttacon/termui"
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
	e.Height = 9
	e.BorderLabelFg = ui.ColorGreen
	e.PaddingTop = 1
	e.PaddingBottom = 1
	e.PaddingLeft = 1
	e.PaddingRight = 1

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
		w.widget.Text = ""

		client := &http.Client{}

		req, err := http.NewRequest("GET", fmt.Sprintf("http://wttr.in/%s?0q", w.location), nil)

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
					bodyStr := string(body)

					if len(bodyStr) == 0 {
						// Error
						w.widget.BorderLabel = "Weather: ERROR"
					} else {
						parts := strings.SplitN(bodyStr, "\n", 3)

						if len(parts) > 0 {
							// Header
							w.widget.BorderLabel = parts[0]

							if len(parts) > 2 {
								// Weather
								w.widget.Text = ConvertANSIToColorStrings(parts[2])
							} else if len(parts) > 1 {
								// Maybe terrible?
								w.widget.Text = ConvertANSIToColorStrings(parts[1])
							}
							w.widget.Text = strings.TrimRight(w.widget.Text, " \t\n\r\x0A")
						} else {
							// Error
							w.widget.BorderLabel = "Weather: ERROR"
						}
					}
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
