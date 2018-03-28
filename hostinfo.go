package main

/**
 * Host information.
 */

import (
	"fmt"
	"strings"
	"time"

	linuxproc "github.com/c9s/goprocinfo/linux"
	ui "github.com/ttacon/termui"
)

////////////////////////////////////////////
// Utility: Time
////////////////////////////////////////////

func getTime() (time.Time, *linuxproc.Uptime) {
	now := time.Now().UTC()
	uptime, err := linuxproc.ReadUptime("/proc/uptime")

	if err != nil {
		uptime = nil
	}

	return now, uptime
}

////////////////////////////////////////////
// Widget: Host Information
////////////////////////////////////////////

type HostInfoWidget struct {
	widget *ui.List
}

func NewHostInfoWidget() *HostInfoWidget {
	// Create base element
	e := ui.NewList()
	e.Height = 5
	e.Border = true
	e.BorderFg = ui.ColorBlue | ui.AttrBold

	// Create widget
	w := &HostInfoWidget{
		widget: e,
	}

	w.update()
	w.resize()

	return w
}

func (w *HostInfoWidget) getGridWidget() ui.GridBufferer {
	return w.widget
}

func (w *HostInfoWidget) update() {
	now, uptime := getTime()
	krbText, krbAttr := getKerberosStatusString()

	// Start building lines
	w.widget.Items = []string{}
	w.widget.PaddingLeft = 2

	// Set time
	w.widget.Items = append(w.widget.Items, fmt.Sprintf("[Time](fg-cyan)....... [%v](fg-magenta)", now.Local().Format("2006/01/02 15:04:05 MST")))

	// Uptime
	w.widget.Items = append(w.widget.Items, fmt.Sprintf("[Uptime](fg-cyan)..... [%v](fg-green)", uptime.GetTotalDuration()))

	// Kerberos
	w.widget.Items = append(w.widget.Items, fmt.Sprintf("[Kerberos](fg-cyan)... [%v](%v)", krbText, krbAttr))
}

func (w *HostInfoWidget) resize() {
	// Do nothing
}

func getKerberosStatusString() (string, string) {
	// Do we have a ticket?
	_, exitCode, _ := execAndGetOutput("klist", nil, "-s")

	hasTicket := exitCode == 0

	// Get the time left
	timeLeftOutput, _, err := execAndGetOutput("kleft", nil, "")
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
	krbText := "Kerberos Ticket"
	krbAttrStr := ""

	if hasTicket {
		if hasTimeLeft {
			krbText = fmt.Sprintf("OK (%v)", timeLeft)
		} else {
			krbText = fmt.Sprintf("OK")
		}
		krbAttrStr = "fg-green,fg-bold"
	} else {
		krbText = fmt.Sprintf("NO TICKET")
		krbAttrStr = "fg-red,fg-bold"
	}

	return krbText, krbAttrStr
}
