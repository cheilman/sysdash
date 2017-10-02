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
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	linuxproc "github.com/c9s/goprocinfo/linux"
	ui "github.com/gizak/termui"
	"github.com/sqp/pulseaudio"
	set "gopkg.in/fatih/set.v0"
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

func rightJustify(width int, str string) string {

	rightJustfyLen := width - utf8.RuneCountInString(str)

	log.Printf("Width: %v, Rune: %v, Len: %v, Justify: %v", width, utf8.RuneCountInString(str), len(str), rightJustfyLen)
	var rightJustify = ""
	if rightJustfyLen > 0 {
		rightJustify = strings.Repeat(" ", rightJustfyLen)
	}

	return rightJustify + str
}

func centerString(width int, str string) string {
	start := (width / 2) - (utf8.RuneCountInString(str) / 2)
	log.Printf("Width: %v, Str: '%v', Start: %v", width, str, start)

	if start > 0 {
		return fmt.Sprintf("%s%s", strings.Repeat(" ", start), str)
	} else {
		return str
	}
}

var ANSI_REGEXP, ANSI_REGEXP_ERR = regexp.Compile(`\x1B\[(([0-9]{1,2})?(;)?([0-9]{1,2})?)?[m,K,H,f,J]`)

func stripANSI(str string) string {
	return ANSI_REGEXP.ReplaceAllLiteralString(str, "")
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

////////////////////////////////////////////
// Widget: Combined Gauges
////////////////////////////////////////////

type GroupedGauges struct {
	ui.Block
	Surround *ui.Par
	Gauges   []ui.Gauge
}

func (s *GroupedGauges) Add(sl ui.Gauge) {
	s.Gauges = append(s.Gauges, sl)
}

func (s *GroupedGauges) Clear() {
	s.Gauges = make([]ui.Gauge, 0)
}

func NewGroupedGauges(surruond *ui.Par, gs ...ui.Gauge) *GroupedGauges {
	gg := &GroupedGauges{
		Surround: surruond,
		Block:    *ui.NewBlock(),
		Gauges:   gs,
	}
	return gg
}

func (gg *GroupedGauges) update() {
	for _, v := range gg.Gauges {
		if v.Border {
			v.Width = gg.Width - 4
		} else {
			v.Width = gg.Width - 2
		}
	}

	// get how many lines gotta display
	h := 0
	for _, v := range gg.Gauges {
		h += v.Height
	}

	if gg.Border {
		gg.Height = h + 2
	} else {
		gg.Height = h
	}

	// Adjust subcomponent X/Y
	startX := gg.X
	startY := gg.Y

	gg.Surround.X = startX
	gg.Surround.Y = startY
	gg.Surround.Width = gg.Width
	gg.Surround.Height = gg.Height

	if gg.Border {
		startX += 1
		startY += 1
	}

	for _, v := range gg.Gauges {
		v.X = startX
		v.Y = startY

		startY += v.Height
	}
}

// Buffer implements Bufferer interface.
func (gg *GroupedGauges) Buffer() ui.Buffer {
	gg.update()

	retval := gg.Surround.Buffer()

	for _, v := range gg.Gauges {
		for dy := 0; dy < v.Height; dy++ {
			for dx := 0; dx < v.Width; dx++ {
				x := v.X + dx
				y := v.Y + dy

				retval.Set(x, y, v.Buffer().At(dx, dy))
			}
		}
	}

	return retval
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

	e.BorderLabel = userHostHeader

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
	e.Height = 1
	e.Border = false
	e.TextFgColor = ui.ColorBlue + ui.AttrBold

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
	now, uptime := getTime()
	//nowStr := now.Local().Format(time.RFC1123Z)
	nowStr := now.Local().Format("2006/01/02 15:04:05 MST")
	uptimeStr := uptime.GetTotalDuration()

	timeStr := fmt.Sprintf("%v -- Up: %v ", nowStr, uptimeStr)

	w.widget.Text = centerString(w.widget.Width, timeStr)
}

func (w *TimeWidget) resize() {
	// Update
	w.update()
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
// Widget: Kerberos
////////////////////////////////////////////

const KerberosUpdateInterval = 10 * time.Second

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
		krbText := "Kerberos Ticket"

		if hasTicket {
			if hasTimeLeft {
				krbText = fmt.Sprintf("Krb: OK (%v)", timeLeft)
			} else {
				krbText = fmt.Sprintf("Krb: OK")
			}
			w.widget.TextFgColor = ui.ColorGreen + ui.AttrBold
		} else {
			krbText = fmt.Sprintf("Krb: NO TICKET")
			w.widget.TextFgColor = ui.ColorRed + ui.AttrBold
		}

		w.widget.Text = centerString(w.widget.Width, krbText)
	}
}

func (w *KerberosWidget) resize() {
	// Do nothing
}

func (w *KerberosWidget) getUpdateInterval() time.Duration {
	return KerberosUpdateInterval
}

func (w *KerberosWidget) getLastUpdated() *time.Time {
	return w.lastUpdated
}

func (w *KerberosWidget) setLastUpdated(t time.Time) {
	w.lastUpdated = &t
}

////////////////////////////////////////////
// Widget: Disk
////////////////////////////////////////////

const DiskUpdateInterval = 30 * time.Second

type DiskWidget struct {
	widget      *GroupedGauges
	lastUpdated *time.Time
	lastUsage   []DiskUsage
}

type DiskUsage struct {
	MountPoint           string
	FSType               string
	TotalSizeInBytes     uint64
	AvailableSizeInBytes uint64
	FreePercentage       float64
	InodesInUse          uint64
	TotalInodes          uint64
	FreeInodesPercentage float64
}

var IgnoreFilesystemTypes = set.New(
	"sysfs", "proc", "udev", "devpts", "tmpfs", "cgroup", "systemd-1",
	"mqueue", "debugfs", "hugetlbfs", "fusectl", "tracefs", "binfmt_misc",
	"devtmpfs", "securityfs", "pstore", "autofs", "fuse.jetbrains-toolbox",
	"fuse.gvfsd-fuse")

func NewDiskWidget() *DiskWidget {
	// Create base element
	surround := ui.NewPar("Disk")
	surround.Border = true

	gg := NewGroupedGauges(surround)

	// Create widget
	w := &DiskWidget{
		widget:      gg,
		lastUpdated: nil,
	}

	w.update()
	w.resize()

	return w
}

func (w *DiskWidget) getGridWidget() ui.GridBufferer {
	return w.widget
}

func (w *DiskWidget) update() {
	if shouldUpdate(w) {
		w.lastUsage = loadDiskUsage()

		log.Printf("DISK: %v", w.lastUsage)

		w.widget.Clear()

		for _, d := range w.lastUsage {
			free := int(100 * d.FreePercentage)
			g := ui.NewGauge()
			g.BorderLabel = d.MountPoint
			g.Height = 3
			g.Percent = free
			g.Label = fmt.Sprintf("Free: %s/%s (%d%%)",
				prettyPrintBytes(d.AvailableSizeInBytes), prettyPrintBytes(d.TotalSizeInBytes), free)
			g.PercentColor = ui.ColorWhite + ui.AttrBold

			if free < 15 {
				g.BarColor = ui.ColorRed + ui.AttrBold
			} else if free < 40 {
				g.BarColor = ui.ColorYellow + ui.AttrBold
			} else if free < 90 {
				g.BarColor = ui.ColorGreen + ui.AttrBold
			} else {
				g.BarColor = ui.ColorCyan + ui.AttrBold
			}

			w.widget.Add(*g)
		}
	}
}

func (w *DiskWidget) resize() {
	// Do nothing
}

func (w *DiskWidget) getUpdateInterval() time.Duration {
	return KerberosUpdateInterval
}

func (w *DiskWidget) getLastUpdated() *time.Time {
	return w.lastUpdated
}

func (w *DiskWidget) setLastUpdated(t time.Time) {
	w.lastUpdated = &t
}

func prettyPrintBytes(bytes uint64) string {
	if bytes > (1024 * 1024 * 1024) {
		gb := float64(bytes) / float64(1024*1024*1024)
		return fmt.Sprintf("%0.2fG", gb)
	} else if bytes > (1024 * 1024) {
		mb := float64(bytes) / float64(1024*1024)
		return fmt.Sprintf("%0.2fM", mb)
	} else if bytes > (1024) {
		kb := float64(bytes) / float64(1024)
		return fmt.Sprintf("%0.2fK", kb)
	} else {
		return fmt.Sprintf("%dbytes", bytes)
	}
}

func loadDiskUsage() []DiskUsage {
	diskUsageData := make([]DiskUsage, 0)

	// Load mount points
	mounts, mountsErr := linuxproc.ReadMounts("/proc/mounts")

	if mountsErr != nil {
		log.Printf("Error loading mounts: %v", mountsErr)
	} else {
		for _, mnt := range mounts.Mounts {

			if IgnoreFilesystemTypes.Has(mnt.FSType) {
				// Skip it
				continue
			}

			// Also skip this docker fs, since it's a dup of root
			if "/var/lib/docker/aufs" == mnt.MountPoint {
				// Skip it
				continue
			}

			statfs := syscall.Statfs_t{}
			statErr := syscall.Statfs(mnt.MountPoint, &statfs)

			if statErr != nil {
				log.Printf("Error statfs-ing mount: %v", mnt.MountPoint)
			} else {
				var totalBytes uint64 = 0
				var availBytes uint64 = 0
				var bytesFreePercent float64 = 0
				var totalInodes uint64 = 0
				var usedInodes uint64 = 0
				var inodesFreePercent float64 = 0

				var blocksize uint64 = 0
				if statfs.Bsize > 0 {
					blocksize = uint64(statfs.Bsize)
				} else {
					blocksize = 1 // bad guess
					log.Printf("Bad block size: %v", statfs.Bsize)
				}

				totalBytes = statfs.Blocks * blocksize
				availBytes = statfs.Bavail * blocksize
				if totalBytes > 0 {
					bytesFreePercent = float64(availBytes) / float64(totalBytes)
					log.Printf("MOUNT: %v -- bytes: %v / %v (%0.2f%%)", mnt.MountPoint, prettyPrintBytes(availBytes), prettyPrintBytes(totalBytes), bytesFreePercent*100)
				} else {
					log.Printf("Bad total bytes: %v", totalBytes)
				}

				totalInodes = statfs.Files
				usedInodes = statfs.Ffree
				if totalInodes > 0 {
					inodesFreePercent = float64(totalInodes-usedInodes) / float64(totalInodes)
					log.Printf("MOUNT: %v -- inodes: %v / %v (%0.0f%%)", mnt.MountPoint, usedInodes, totalInodes, inodesFreePercent*100)
				} else {
					log.Printf("Bad total inodes: %v", totalInodes)
				}

				usage := DiskUsage{
					MountPoint:           mnt.MountPoint,
					FSType:               mnt.FSType,
					TotalSizeInBytes:     totalBytes,
					AvailableSizeInBytes: availBytes,
					FreePercentage:       bytesFreePercent,
					TotalInodes:          totalInodes,
					InodesInUse:          usedInodes,
					FreeInodesPercentage: inodesFreePercent,
				}

				log.Printf("MOUNT: %v -- %v", mnt.MountPoint, usage)

				diskUsageData = append(diskUsageData, usage)
			}
		}
	}

	return diskUsageData
}

////////////////////////////////////////////
// Widget: CPU
////////////////////////////////////////////

type CPUWidget struct {
	widget *ui.LineChart

	lastStat     linuxproc.CPUStat
	curStat      linuxproc.CPUStat
	cpuPercent   float64
	loadLast1Min []float64
	loadLast5Min []float64
}

func NewCPUWidget() *CPUWidget {
	// Create base element
	e := ui.NewLineChart()
	e.Height = 20
	e.Border = true

	// Create widget
	w := &CPUWidget{
		widget:       e,
		cpuPercent:   0,
		loadLast1Min: make([]float64, 0),
		loadLast5Min: make([]float64, 0),
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

	w.widget.BorderLabel = fmt.Sprintf("CPU: %0.2f%%", w.cpuPercent*100)
	w.widget.Data = w.loadLast1Min
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
		// Record, keep a fixed number around
		if len(w.loadLast1Min) > w.widget.Width {
			w.loadLast1Min = append(w.loadLast1Min[1:], loadavg.Last1Min)
		} else {
			w.loadLast1Min = append(w.loadLast1Min, loadavg.Last1Min)
		}

		if len(w.loadLast5Min) > w.widget.Width {
			w.loadLast5Min = append(w.loadLast5Min[1:], loadavg.Last5Min)
		} else {
			w.loadLast5Min = append(w.loadLast5Min, loadavg.Last5Min)
		}
	}
}

////////////////////////////////////////////
// Widget: Battery
////////////////////////////////////////////

const BatteryUpdateIntervalSeconds = 10

type BatteryWidget struct {
	widget      *ui.Gauge
	lastUpdated *time.Time
}

func NewBatteryWidget() *BatteryWidget {
	// Create base element
	e := ui.NewGauge()
	e.Height = 3
	e.Border = true

	// Create widget
	w := &BatteryWidget{
		widget:      e,
		lastUpdated: nil,
	}

	w.update()
	w.resize()

	return w
}

func (w *BatteryWidget) getGridWidget() ui.GridBufferer {
	return w.widget
}

func (w *BatteryWidget) update() {
	if shouldUpdate(w) {
		// Load battery info
		output, _, err := execAndGetOutput("ibam-battery-prompt", "-p")

		if err == nil {
			// Parse the output
			lines := strings.Split(output, "\n")
			if len(lines) >= 4 {
				// we have enough
				timeLeft := stripANSI(lines[1])
				isCharging, chargeErr := strconv.ParseBool(lines[2])
				batteryPercent, percentErr := strconv.Atoi(lines[4])

				if chargeErr != nil {
					isCharging = false
					log.Printf("Error reading charge status: '%v' -- %v", lines[2], chargeErr)
				}

				if percentErr != nil {
					batteryPercent = 0
					log.Printf("Error reading battery percent: '%v' -- %v", lines[4], chargeErr)
				}

				var battColor = ui.ColorBlue

				if batteryPercent > 80 {
					battColor = ui.ColorGreen
				} else if batteryPercent > 20 {
					battColor = ui.ColorMagenta
				} else {
					battColor = ui.ColorRed
				}

				if isCharging {
					w.widget.BorderLabel = "Battery (charging)"
					w.widget.BorderLabelFg = ui.ColorCyan + ui.AttrBold
				} else {
					w.widget.BorderLabel = "Battery"
					w.widget.BorderLabelFg = battColor + ui.AttrBold
				}

				w.widget.Percent = batteryPercent
				w.widget.BarColor = battColor + ui.AttrBold
				w.widget.Label = fmt.Sprintf("%d%% (%s)", batteryPercent, timeLeft)
				w.widget.LabelAlign = ui.AlignRight
				w.widget.PercentColor = ui.ColorWhite + ui.AttrBold
				//w.widget.PercentColorHighlighted = ui.ColorBlack
				w.widget.PercentColorHighlighted = w.widget.PercentColor
			} else {
				log.Printf("Not enough lines from battery command!  Output: %v", output)
			}
		} else {
			log.Printf("Error executing battery command: %v", err)
		}
	}
}

func (w *BatteryWidget) resize() {
	// Do nothing
}

func (w *BatteryWidget) getUpdateInterval() time.Duration {
	// Update every 10 seconds
	return time.Second * 10
}

func (w *BatteryWidget) getLastUpdated() *time.Time {
	return w.lastUpdated
}

func (w *BatteryWidget) setLastUpdated(t time.Time) {
	w.lastUpdated = &t
}

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
		// TODO: Add events for update (and speed up our update interval)

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

////////////////////////////////////////////
// Where the real stuff happens
////////////////////////////////////////////

func main() {

	if ANSI_REGEXP_ERR != nil {
		panic(ANSI_REGEXP_ERR)
	}

	// Set up logging
	logFile, logErr := os.OpenFile("go.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0660)
	if logErr != nil {
		panic(logErr)
	}
	defer logFile.Close()

	log.SetOutput(logFile)

	// Set up the console UI
	uiErr := ui.Init()
	if uiErr != nil {
		panic(uiErr)
	}
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

	battery := NewBatteryWidget()
	widgets = append(widgets, battery)

	audio := NewAudioWidget()
	widgets = append(widgets, audio)

	disk := NewDiskWidget()
	widgets = append(widgets, disk)

	cpu := NewCPUWidget()
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
			ui.NewCol(8, 0, time.getGridWidget()),
			ui.NewCol(4, 0, kerberos.getGridWidget())),
		ui.NewRow(
			ui.NewCol(6, 0, network.getGridWidget()),
			ui.NewCol(6, 0, disk.getGridWidget())),
		ui.NewRow(
			ui.NewCol(6, 0, battery.getGridWidget()),
			ui.NewCol(6, 0, audio.getGridWidget())),
		ui.NewRow(
			ui.NewCol(12, 0, cpu.getGridWidget())),
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
