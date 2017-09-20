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

////////////////////////////////////////////
// Utility: Formatting
////////////////////////////////////////////

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

	if fillRunes < 0 {
		fillRunes = 0
	}

	fillStr := strings.Repeat(fillChar, fillRunes)

	return fmt.Sprintf("%s %s %s", left, fillStr, right)
}

////////////////////////////////////////////
// Utility: Data gathering
////////////////////////////////////////////

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

////////////////////////////////////////////
// Utility: Widgets
////////////////////////////////////////////

type CAHWidget interface {
	getGridWidget() ui.GridBufferer
	update(timer uint64)
	resize()
}

type TempWidget struct {
	widget *ui.Par
}

func NewTempWidget(l string) *TempWidget {
	p := ui.NewPar(l)
	p.Height = 5
	p.BorderLabel = l

	// Create our widget
	w := &TempWidget{
		widget: p,
	}

	// Invoke its functions
	w.update(0)
	w.resize()

	return w
}

func (w *TempWidget) getGridWidget() ui.GridBufferer {
	return w.widget
}

func (w *TempWidget) update(count uint64) {
	// Do nothing
}

func (w *TempWidget) resize() {
	// Do nothing
}

////////////////////////////////////////////
// Widget: Header
////////////////////////////////////////////

type HeaderWidget struct {
	widget         *ui.Par
	userHostHeader string
}

func NewHeaderWidget() *HeaderWidget {
	// Create base element
	e := ui.NewPar("")

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

	// Create our widget
	w := &HeaderWidget{
		widget:         e,
		userHostHeader: userHostHeader,
	}

	// Invoke its functions
	w.update(0)
	w.resize()

	return w
}

func (w *HeaderWidget) getGridWidget() ui.GridBufferer {
	return w.widget
}

func (w *HeaderWidget) update(count uint64) {
	now, uptime := getTime()
	nowStr := now.Local().Format(time.RFC1123Z)
	uptimeStr := uptime.GetTotalDuration()

	timeStr := fmt.Sprintf("%v (%v) ", nowStr, uptimeStr)

	w.widget.BorderLabel = fitAStringToWidth(ui.TermWidth()-4, w.userHostHeader, timeStr, "-")
}

func (w *HeaderWidget) resize() {
	// Update header on window resize
	w.widget.X = 0
	w.widget.Y = 0
	w.widget.Width = ui.TermWidth()
	w.widget.Height = ui.TermHeight()

	// Also update dynamic information, since the spacing depends on the window width
	w.update(0)
}

////////////////////////////////////////////
// Widget: Time
////////////////////////////////////////////

type TimeWidget struct {
	widget      *ui.Par
	tickCounter uint64
}

func NewTimeWidget() *TimeWidget {
	// Create base element
	e := ui.NewPar("Time")
	e.Height = 4

	// Create widget
	w := &TimeWidget{
		widget:      e,
		tickCounter: 0,
	}

	w.update(0)
	w.resize()

	return w
}

func (w *TimeWidget) getGridWidget() ui.GridBufferer {
	return w.widget
}

func (w *TimeWidget) update(count uint64) {
	w.widget.BorderLabel = fmt.Sprintf("Time (%v)", w.tickCounter)
	w.tickCounter++

	now, uptime := getTime()
	nowStr := now.Format(time.RFC1123Z)
	uptimeStr := uptime.GetTotalDuration()

	w.widget.Text = fmt.Sprintf("Now: %v\nUptime: %v", nowStr, uptimeStr)
}

func (w *TimeWidget) resize() {
	// Do nothing
}

////////////////////////////////////////////
// Creating our widgets
////////////////////////////////////////////

//
//func makeNetwork() *CAHWidget {
//	// Create container
//	c := ui.NewPar("Networking")
//
//	// Create widget
//	w := ui.NewTable()
//	w.Height = 8
//	w.Border = false
//
//	var lastCount uint64 = 0
//
//	// Function for dynamic information
//	f := func(count uint64) {
//		if (count == 0) || ((count - lastCount) >= 30000) {
//			// First try, or after a period of time
//			lastCount = count
//
//			// Load network interfaces and information
//			rows := [][]string{
//				[]string{"interface0", "interface2", "interface3"},
//				[]string{"123.456.789.123", "123.456.789.123", "123.456.789.123"},
//			}
//
//			w.Rows = rows
//
//			w.Analysis()
//			w.SetSize()
//		}
//	}
//
//	// Function for resizes
//	r := func() {
//		c.X = w.X
//		c.Y = w.Y
//		c.Width = w.Width
//		c.Height = w.Height
//	}
//
//	// Load dynamic info
//	f(0)
//	r()
//
//	return NewCAHWidget(w, &f, &r)
//}
//
//func makeBattAudio() *CAHWidget {
//	// Create widget
//	w := ui.NewPar("")
//	w.Height = 3
//
//	// Static information
//	curUser, userErr := user.Current()
//	userName := "unknown"
//	if userErr == nil {
//		userName = curUser.Username
//	}
//
//	hostName, hostErr := os.Hostname()
//	if hostErr != nil {
//		hostName = "unknown"
//	}
//
//	prettyName, prettyNameErr := execAndGetOutput("pretty-hostname")
//
//	if (prettyNameErr == nil) && (prettyName != hostName) {
//		// Host/pretty name are different
//		w.BorderLabel = fmt.Sprintf(" %v @ %v (%v) ", userName, prettyName, hostName)
//	} else {
//		// Host/pretty name are the same (or pretty failed)
//		w.BorderLabel = fmt.Sprintf(" %v @ %v ", userName, hostName)
//	}
//
//	// Function for dynamic information
//	f := func(count uint64) {
//		if count == 0 {
//			// Invoked at startup
//		} else {
//			// Invoked on timer
//		}
//	}
//
//	// Load dynamic info
//	f(0)
//
//	return NewCAHWidget(w, &f, nil)
//}

////////////////////////////////////////////
// Where the real stuff happens
////////////////////////////////////////////

func main() {
	// Set up the console UI
	err := ui.Init()
	if err != nil {
		panic(err)
	}
	defer log.Printf("Final Timer: %v (%v)", timerCounter, lastTimer)
	defer ui.Close()

	// New 5-second timer
	ui.DefaultEvtStream.Merge("timer", ui.NewTimerCh(5*time.Second))

	//
	// Create the widgets
	//
	widgets := make([]CAHWidget, 0)

	header := NewHeaderWidget()
	widgets = append(widgets, header)

	network := NewTempWidget("network")
	widgets = append(widgets, network)

	time := NewTimeWidget()
	widgets = append(widgets, time)

	battAudio := NewTempWidget("battery/audio")
	widgets = append(widgets, battAudio)

	disk := NewTempWidget("disk")
	widgets = append(widgets, disk)

	cpu := NewTempWidget("cpu")
	widgets = append(widgets, cpu)

	repo := NewTempWidget("repos")
	widgets = append(widgets, repo)

	commits := NewTempWidget("commits")
	widgets = append(widgets, commits)

	twitter1 := NewTempWidget("tinycare")
	widgets = append(widgets, twitter1)

	twitter2 := NewTempWidget("selfcare")
	widgets = append(widgets, twitter2)

	twitter3 := NewTempWidget("a strange voyage")
	widgets = append(widgets, twitter3)

	weather := NewTempWidget("weather")
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
			ui.NewCol(3, 0, network.getGridWidget()),
			ui.NewCol(3, 0, disk.getGridWidget()),
			ui.NewCol(3, 0, cpu.getGridWidget()),
			ui.NewCol(3, 0, battAudio.getGridWidget())),
		ui.NewRow(
			ui.NewCol(12, 0, time.getGridWidget())),
		ui.NewRow(
			ui.NewCol(6, 0, repo.getGridWidget()),
			ui.NewCol(6, 0, commits.getGridWidget())),
		ui.NewRow(
			ui.NewCol(3, 0, weather.getGridWidget()),
			ui.NewCol(3, 0, twitter1.getGridWidget()),
			ui.NewCol(3, 0, twitter2.getGridWidget()),
			ui.NewCol(3, 0, twitter3.getGridWidget())))

	ui.Body.Align()

	render := func() {
		ui.Body.Align()
		ui.Clear()
		ui.Render(header.widget, ui.Body)
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

		// Call all update funcs
		for _, w := range widgets {
			w.update(i)
		}

		// Re-render
		render()
	})

	ui.Handle("/sys/wnd/resize", func(ui.Event) {
		// Re-layout on resize
		ui.Body.Width = ui.TermWidth() - 2

		// Call all resize funcs
		for _, w := range widgets {
			w.resize()
		}

		// Re-render
		render()
	})

	ui.Loop()
}
