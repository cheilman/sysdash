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
	"syscall"
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

func execAndGetOutput(name string, args ...string) (stdout string, exitCode int, err error) {
	cmd := exec.Command(name, args...)

	var out bytes.Buffer
	cmd.Stdout = &out

	err = cmd.Run()

	// Getting the exit code is platform dependant, this code isn't portable
	exitCode = 0

	if err != nil {
		// Based on: https://stackoverflow.com/questions/10385551/get-exit-code-go
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()
		} else {
			// Failed, but on a platform where this conversion doesn't work...
			exitCode = 1
		}
	}

	stdout = out.String()

	return
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

	prettyName, _, prettyNameErr := execAndGetOutput("pretty-hostname")

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
	update()
	resize()
}

type UpdateInterval interface {
	getUpdateInterval() time.Duration
	getLastUpdated() *time.Time
	setLastUpdated(t time.Time)
}

func shouldUpdate(updater UpdateInterval) bool {
	now := time.Now()
	lastUpdated := updater.getLastUpdated()

	if lastUpdated == nil {
		// First time, execute it
		updater.setLastUpdated(now)
		return true
	} else {
		// Has enough time passed?
		elapsed := now.Sub(*lastUpdated)

		if elapsed.Nanoseconds() > updater.getUpdateInterval().Nanoseconds() {
			updater.setLastUpdated(time.Now())
			return true
		}
	}

	return false
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
	w.update()
	w.resize()

	return w
}

func (w *TempWidget) getGridWidget() ui.GridBufferer {
	return w.widget
}

func (w *TempWidget) update() {
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
	lastUpdated    uint64
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
	w.update()
	w.resize()

	return w
}

func (w *HeaderWidget) getGridWidget() ui.GridBufferer {
	return w.widget
}

func (w *HeaderWidget) update() {
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
	w.update()
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

	w.update()
	w.resize()

	return w
}

func (w *TimeWidget) getGridWidget() ui.GridBufferer {
	return w.widget
}

func (w *TimeWidget) update() {
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
// Widget: Kerberos
////////////////////////////////////////////

const KerberosUpdateIntervalSeconds = 10

type KerberosWidget struct {
	widget      *ui.Par
	lastUpdated *time.Time
}

func NewKerberosWidget() *KerberosWidget {
	// Create base element
	e := ui.NewPar("Kerberos")
	e.Height = 1
	e.Border = false

	// Create widget
	w := &KerberosWidget{
		widget:      e,
		lastUpdated: nil,
	}

	w.update()
	w.resize()

	return w
}

func (w *KerberosWidget) getGridWidget() ui.GridBufferer {
	return w.widget
}

func (w *KerberosWidget) update() {
	if shouldUpdate(w) {
		// Do we have a ticket?
		_, exitCode, _ := execAndGetOutput("klist", "-s")

		hasTicket := exitCode == 0

		// Get the time left
		timeLeftOutput, _, err := execAndGetOutput("kleft", "")
		var hasTimeLeft = false
		var timeLeft string

		if err == nil {
			timeLeftParts := strings.Split(timeLeftOutput, " ")
			if len(timeLeftParts) > 1 {
				hasTimeLeft = true
				timeLeft = strings.TrimSpace(timeLeftParts[1])
			}
		}

		// Piece it all together
		if hasTicket {
			if hasTimeLeft {
				w.widget.Text = fmt.Sprintf("Krb: OK (%v)", timeLeft)
			} else {
				w.widget.Text = fmt.Sprintf("Krb: OK")
			}
			w.widget.TextFgColor = ui.ColorGreen + ui.AttrBold
		} else {
			w.widget.Text = fmt.Sprintf("Krb: NO TICKET")
			w.widget.TextFgColor = ui.ColorRed + ui.AttrBold
		}
	}
}

func (w *KerberosWidget) resize() {
	// Do nothing
}

func (w *KerberosWidget) getUpdateInterval() time.Duration {
	// Update every 10 seconds
	return time.Second * 5
}

func (w *KerberosWidget) getLastUpdated() *time.Time {
	return w.lastUpdated
}

func (w *KerberosWidget) setLastUpdated(t time.Time) {
	w.lastUpdated = &t
}

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

	ui.DefaultEvtStream.Merge("timer", ui.NewTimerCh(5*time.Second))

	//
	// Create the widgets
	//
	widgets := make([]CAHWidget, 0)

	header := NewHeaderWidget()
	widgets = append(widgets, header)

	kerberos := NewKerberosWidget()
	widgets = append(widgets, kerberos)

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
			ui.NewCol(6, 0, time.getGridWidget()),
			ui.NewCol(6, 0, kerberos.getGridWidget())),
		ui.NewRow(
			ui.NewCol(3, 0, network.getGridWidget()),
			ui.NewCol(3, 0, disk.getGridWidget()),
			ui.NewCol(3, 0, cpu.getGridWidget()),
			ui.NewCol(3, 0, battAudio.getGridWidget())),
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
		// Call all update funcs
		for _, w := range widgets {
			w.update()
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
