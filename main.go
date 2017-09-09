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
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"

	ui "github.com/gizak/termui"
)

func makeP(l string) *ui.Par {
	p := ui.NewPar(l)
	p.Height = 5
	p.BorderLabel = l

	return p
}

func execAndGetOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()

	return out.String(), err
}

func getUsername() string {
	curUser, userErr := user.Current()
	userName := "unknown"
	if userErr == nil {
		userName = curUser.Username
	}

	return userName
}

func getHostname() (string, string) {
	hostName, hostErr := os.Hostname()
	if hostErr != nil {
		hostName = "unknown"
	}

	prettyName, prettyNameErr := execAndGetOutput("pretty-hostname")

	if prettyNameErr == nil {
		return hostName, prettyName
	} else {
		return hostName, hostName
	}
}

// Header: User @ hostname
func makeHeader() (*ui.Par, func(*ui.EvtTimer)) {
	// Create widget
	w := ui.NewPar("")
	w.Height = 3

	// Static information
	userName := getUsername()
	hostName, prettyName := getHostname()

	if prettyName != hostName {
		// Host/pretty name are different
		w.BorderLabel = fmt.Sprintf(" %v @ %v (%v) ", userName, prettyName, hostName)
	} else {
		// Host/pretty name are the same (or pretty failed)
		w.BorderLabel = fmt.Sprintf(" %v @ %v ", userName, hostName)
	}

	// Function for dynamic information
	f := func(e *ui.EvtTimer) {
		if e == nil {
			// Invoked at startup
		} else {
			// Invoked on timer
		}
	}

	// Load dynamic info
	f(nil)

	return w, f
}

func makeNetwork() (*ui.Par, *ui.Table, func(*ui.EvtTimer), func()) {
	// Create container
	c := ui.NewPar("Networking")

	// Create widget
	w := ui.NewTable()
	w.Height = 8
	w.Border = false

	var lastCount uint64 = 0

	// Function for dynamic information
	f := func(e *ui.EvtTimer) {
		if (e == nil) || ((e.Count - lastCount) >= 30000) {
			// First try, or after a period of time
			if e != nil {
				lastCount = e.Count
			}

			// Load network interfaces and information
			rows := [][]string{
				[]string{"interface0", "interface2", "interface3"},
				[]string{"123.456.789.123", "123.456.789.123", "123.456.789.123"},
			}

			w.Rows = rows

			w.Analysis()
			w.SetSize()
		}
	}

	// Function for resizes
	r := func() {
		c.X = w.X
		c.Y = w.Y
		c.Width = w.Width
		c.Height = w.Height
	}

	// Load dynamic info
	f(nil)
	r()

	return c, w, f, r
}

func makeBattAudio() (*ui.Par, func(*ui.EvtTimer)) {
	// Create widget
	w := ui.NewPar("")
	w.Height = 3

	// Static information
	curUser, userErr := user.Current()
	userName := "unknown"
	if userErr == nil {
		userName = curUser.Username
	}

	hostName, hostErr := os.Hostname()
	if hostErr != nil {
		hostName = "unknown"
	}

	prettyName, prettyNameErr := execAndGetOutput("pretty-hostname")

	if (prettyNameErr == nil) && (prettyName != hostName) {
		// Host/pretty name are different
		w.BorderLabel = fmt.Sprintf(" %v @ %v (%v) ", userName, prettyName, hostName)
	} else {
		// Host/pretty name are the same (or pretty failed)
		w.BorderLabel = fmt.Sprintf(" %v @ %v ", userName, hostName)
	}

	// Function for dynamic information
	f := func(e *ui.EvtTimer) {
		if e == nil {
			// Invoked at startup
		} else {
			// Invoked on timer
		}
	}

	// Load dynamic info
	f(nil)

	return w, f
}

func main() {
	// Set up the console UI
	err := ui.Init()
	if err != nil {
		panic(err)
	}
	defer ui.Close()

	//
	// Create the widgets
	//

	header, headerFunc := makeHeader()
	networkContainer, network, networkFunc, networkResize := makeNetwork()

	battAudio := makeP("battery/audio")
	disk := makeP("disk")
	cpu := makeP("cpu")
	repo := makeP("repos")
	commits := makeP("commits")
	twitter1 := makeP("tinycare")
	twitter2 := makeP("selfcare")
	twitter3 := makeP("a strange voyage")
	weather := makeP("weather")

	//
	// Create the layout
	//

	// Header box
	header.X = 0
	header.Y = 0
	header.Width = ui.TermWidth()
	header.Height = ui.TermHeight()

	// Allow the header box to wrap all around
	ui.Body.Width = ui.TermWidth() - 2
	ui.Body.X = 1
	ui.Body.Y = 1

	ui.Body.AddRows(
		ui.NewRow(
			ui.NewCol(3, 0, network),
			ui.NewCol(3, 0, disk),
			ui.NewCol(3, 0, cpu),
			ui.NewCol(3, 0, battAudio)),
		ui.NewRow(
			ui.NewCol(6, 0, repo),
			ui.NewCol(6, 0, commits)),
		ui.NewRow(
			ui.NewCol(3, 0, weather),
			ui.NewCol(3, 0, twitter1),
			ui.NewCol(3, 0, twitter2),
			ui.NewCol(3, 0, twitter3)))

	ui.Body.Align()

	//
	// Timer Updates
	//

	timerFunc := func(e ui.Event) {
		pt := e.Data.(ui.EvtTimer)
		t := &pt

		// Call all update funcs
		headerFunc(t)
		networkFunc(t)
	}

	//
	//  Activate
	//

	ui.Render(header, ui.Body)

	ui.Handle("/sys/kbd/q", func(ui.Event) {
		// press q to quit
		ui.StopLoop()
	})

	ui.Handle("/sys/kbd/C-c", func(ui.Event) {
		// ctrl-c to quit
		ui.StopLoop()
	})

	ui.Handle("/timer/5s", timerFunc)

	ui.Handle("/sys/wnd/resize", func(e ui.Event) {
		// Update header on resize
		header.Width = ui.TermWidth()
		header.Height = ui.TermHeight()

		// Re-layout on resize
		ui.Body.Width = ui.TermWidth() - 2
		ui.Body.Align()
		ui.Clear()
		ui.Render(header, networkContainer, ui.Body)

		// Update resize funcs
		networkResize()
	})

	ui.Loop()
}
