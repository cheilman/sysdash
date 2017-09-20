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
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"time"
	"unicode/utf8"

	linuxproc "github.com/c9s/goprocinfo/linux"
	ui "github.com/gizak/termui"
)

var timerCounter uint64 = 0
var lastTimer uint64 = 0

/**
 * Make a string as wide as requested, with stuff left justified and right justified.
 *
 * width:       How wide to get.
 * left:        What text goes on the left?
 * right:       What text goes on the right?
 * fillChar:    What character to use as the filler.
 */
func fitAStringToWidth(width int, left string, right string, fillChar string) string {
	leftLen := utf8.RuneCountInString(left)
	rightLen := utf8.RuneCountInString(right)
	fillCharLen := utf8.RuneCountInString(fillChar) // Usually 1

	// Figure out how many filler chars we need
	fillLen := width - (leftLen + rightLen)
	fillRunes := (fillLen - 1 + fillCharLen) / fillCharLen
	fillStr := strings.Repeat(fillChar, fillRunes)

	return fmt.Sprintf("%s %s %s", left, fillStr, right)
}

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

func getTime() (time.Time, *linuxproc.Uptime) {
	now := time.Now().UTC()
	uptime, err := linuxproc.ReadUptime("/proc/uptime")

	if err != nil {
		uptime = nil
	}

	return now, uptime

}

// Header: User @ hostname
func makeHeader() (*ui.Par, func(uint64)) {
	// Create widget
	w := ui.NewPar("")
	w.Height = 3

	// Static information
	userName := getUsername()
	hostName, prettyName := getHostname()
	var userHostHeader string

	if prettyName != hostName {
		// Host/pretty name are different
		userHostHeader = fmt.Sprintf(" %v @ %v (%v)", userName, prettyName, hostName)
	} else {
		// Host/pretty name are the same (or pretty failed)
		userHostHeader = fmt.Sprintf("%v @ %v", userName, hostName)
	}

	// Function for dynamic information
	f := func(count uint64) {
		now, uptime := getTime()
		nowStr := now.Format(time.RFC1123Z)
		uptimeStr := uptime.GetTotalDuration()

		timeStr := fmt.Sprintf("%v (%v) ", nowStr, uptimeStr)

		w.BorderLabel = fitAStringToWidth(ui.TermWidth()-4, userHostHeader, timeStr, "-")
	}

	// Load dynamic info
	f(0)

	return w, f
}

func makeNetwork() (*ui.Par, *ui.Table, func(uint64), func()) {
	// Create container
	c := ui.NewPar("Networking")

	// Create widget
	w := ui.NewTable()
	w.Height = 8
	w.Border = false

	var lastCount uint64 = 0

	// Function for dynamic information
	f := func(count uint64) {
		if (count == 0) || ((count - lastCount) >= 30000) {
			// First try, or after a period of time
			lastCount = count

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
	f(0)
	r()

	return c, w, f, r
}

func makeTime() (*ui.Par, func(uint64)) {
	// Create widget
	w := ui.NewPar("Time")
	w.Height = 4

	c := 1

	// Function for dynamic information
	f := func(count uint64) {
		w.BorderLabel = fmt.Sprintf("Time (%v)", c)
		c = c + 1

		now, uptime := getTime()
		nowStr := now.Format(time.RFC1123Z)
		uptimeStr := uptime.GetTotalDuration()

		w.Text = fmt.Sprintf("Now: %v\nUptime: %v", nowStr, uptimeStr)
	}

	// Load dynamic info
	f(0)

	return w, f
}

func makeBattAudio() (*ui.Par, func(uint64)) {
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
	f := func(count uint64) {
		if count == 0 {
			// Invoked at startup
		} else {
			// Invoked on timer
		}
	}

	// Load dynamic info
	f(0)

	return w, f
}

func main() {
	// Set up the console UI
	err := ui.Init()
	if err != nil {
		panic(err)
	}
	defer log.Printf("Final Timer: %v (%v)", timerCounter, lastTimer)
	defer ui.Close()

	ui.DefaultEvtStream.Merge("timer", ui.NewTimerCh(5*time.Second))

	//
	// Create the widgets
	//

	header, headerFunc := makeHeader()
	//networkContainer, network, networkFunc, networkResize := makeNetwork()
	_, network, networkFunc, networkResize := makeNetwork()
	time, timeFunc := makeTime()

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
			ui.NewCol(12, 0, time)),
		ui.NewRow(
			ui.NewCol(6, 0, repo),
			ui.NewCol(6, 0, commits)),
		ui.NewRow(
			ui.NewCol(3, 0, weather),
			ui.NewCol(3, 0, twitter1),
			ui.NewCol(3, 0, twitter2),
			ui.NewCol(3, 0, twitter3)))

	ui.Body.Align()

	render := func() {
		ui.Body.Align()
		ui.Clear()
		//ui.Render(header, networkContainer, ui.Body)
		ui.Render(header, ui.Body)
	}

	//
	//  Activate
	//

	log.Printf("Failed: %v", timerCounter)

	render()

	ui.Handle("/sys/kbd/q", func(ui.Event) {
		// press q to quit
		ui.StopLoop()
	})

	ui.Handle("/sys/kbd/C-c", func(ui.Event) {
		// ctrl-c to quit
		ui.StopLoop()
	})

	ui.Handle("/timer/5s", func(e ui.Event) {
		t := e.Data.(ui.EvtTimer)
		i := t.Count

		timerCounter++
		lastTimer = i

		log.Printf("Timer: %v (%v)", timerCounter, lastTimer)

		// Call all update funcs
		headerFunc(lastTimer)
		networkFunc(lastTimer)
		timeFunc(lastTimer)

		// Re-render
		render()
	})

	ui.Handle("/sys/wnd/resize", func(ui.Event) {
		// Update header on resize
		header.Width = ui.TermWidth()
		header.Height = ui.TermHeight()

		// Re-layout on resize
		ui.Body.Width = ui.TermWidth() - 2

		// Update resize funcs
		networkResize()

		// Re-render
		render()
	})

	ui.Loop()
}
