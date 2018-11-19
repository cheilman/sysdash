package main

/*
 * A shiny status page.
 *
 * Want it to combine my existing idle page and tiny-care-terminal.
 *
 * Things to include:
 *  - some twitter accounts
 *      - @tinycarebot, @selfcare_bot and @magicrealismbot. Maybe that boat one instead of magic realism.
 *  - weather
 *  - recent git commits
 *  - system status:
 *      - User/hostname
 *      - Kerberos ticket status
 *      - Current time
 *      - Uptime
 *      - Battery and time left
 *      - Audio status and volume
 *      - Network
 *          - Local, docker, wireless
 *      - Disk
 *          - Mounts, free/used/total/percentage w/ color
 *      - CPU
 *          - Load average w/ color
 *          - Percentage
 *          - Top processes?
 *      - Status of git repos
 *
 * Minimum terminal size to support:
 *  - 189x77 (half monitor with some stacks)
 *  - 104x56ish? (half the laptop screen with some stacks)
 *  - 100x50? (nice and round)
 *  - 80x40? (my default putty)
 *
 */

import (
	"io/ioutil"
	"log"
	"os"
	"time"

	ui "github.com/gizak/termui"
)

////////////////////////////////////////////
// Where the real stuff happens
////////////////////////////////////////////

func main() {
	// Set up logging?
	if LogToFile() {
		logFile, logErr := os.OpenFile("go.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0660)
		if logErr != nil {
			panic(logErr)
		}
		defer logFile.Close()

		log.SetOutput(logFile)
	} else {
		// Disable logging
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}

	// Set up the console UI
	uiErr := ui.Init()
	if uiErr != nil {
		panic(uiErr)
	}
	defer ui.Close()

	//
	// Create the widgets
	//
	widgets := make([]CAHWidget, 0)

	header := NewHeaderWidget()
	widgets = append(widgets, header)

	hostInfo := NewHostInfoWidget()
	widgets = append(widgets, hostInfo)

	network := NewNetworkWidget()
	widgets = append(widgets, network)

	battery := NewBatteryWidget()
	widgets = append(widgets, battery)

	audio := NewAudioWidget()
	widgets = append(widgets, audio)

	disk := NewDiskColumn(6, 0)
	widgets = append(widgets, disk)

	cpu := NewCPUWidget()
	widgets = append(widgets, cpu)

	repo := NewGitRepoWidget()
	widgets = append(widgets, repo)

	twitter1 := NewTwitterWidget(GetTwitterAccount1(), ui.ColorBlue|ui.AttrBold)
	widgets = append(widgets, twitter1)

	twitter2 := NewTwitterWidget(GetTwitterAccount2(), ui.ColorCyan)
	widgets = append(widgets, twitter2)

	twitter3 := NewTwitterWidget(GetTwitterAccount3(), ui.ColorMagenta)
	widgets = append(widgets, twitter3)

	weather := NewWeatherWidget(GetWeatherLocation())
	widgets = append(widgets, weather)

	//
	// Create the layout
	//

	// Give space around the ui.Body for the header box to wrap all around
	ui.Body.Width = ui.TermWidth() - 2
	ui.Body.X = 1
	ui.Body.Y = 1

	ui.Body.AddRows(
		ui.NewRow(
			ui.NewCol(6, 0, hostInfo.getGridWidget(), battery.getGridWidget(), audio.getGridWidget(), weather.getGridWidget()),
			ui.NewCol(6, 0, cpu.getGridWidget())),
		ui.NewRow(
			disk.getColumn(),
			ui.NewCol(6, 0, network.getGridWidget())),
		ui.NewRow(
			ui.NewCol(12, 0, repo.getGridWidget())),
		ui.NewRow(
			ui.NewCol(4, 0, twitter1.getGridWidget()),
			ui.NewCol(4, 0, twitter2.getGridWidget()),
			ui.NewCol(4, 0, twitter3.getGridWidget())))

	ui.Body.Align()

	render := func() {
		ui.Body.Align()
		ui.Clear()
		ui.Render(header.widget, ui.Body)
	}

	//
	//  Activate
	//

	render()

	ui.Handle("q", func(ui.Event) {
		// press q to quit
		ui.StopLoop()
	})

	ui.Handle("C-c", func(ui.Event) {
		// ctrl-c to quit
		ui.StopLoop()
	})

	firstTimeResize := false
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			// Call all update funcs
			for _, w := range widgets {
				w.update()
			}

			// Call all resize funcs (only the first time)
			if !firstTimeResize {
				firstTimeResize = true
				for _, w := range widgets {
					w.resize()
				}
			}

			// Re-render
			render()
			<-ticker.C
		}
	}()

	ui.Handle("<Resize>", func(e ui.Event) {
		payload := e.Payload.(ui.Resize)

		// Re-layout on resize
		ui.Body.Width = payload.Width - 2

		// Call all resize funcs
		for _, w := range widgets {
			w.resize()
		}

		// Re-render
		render()
	})

	ui.Loop()
}
