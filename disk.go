package main

/**
 * Disk Usage
 */

import (
	"fmt"
	"log"
	"sort"
	"syscall"
	"time"

	linuxproc "github.com/c9s/goprocinfo/linux"
	ui "github.com/gizak/termui"
	set "gopkg.in/fatih/set.v0"
)

////////////////////////////////////////////
// Utility: Disk Usage
////////////////////////////////////////////

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
	"fuse.gvfsd-fuse", "fuse.lxcfs")

func loadDiskUsage() map[string]DiskUsage {
	diskUsageData := make(map[string]DiskUsage, 0)

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

			// Also skip these docker fs, since it's a dup of root
			if "/var/lib/docker/aufs" == mnt.MountPoint || "/var/lib/docker/devicemapper" == mnt.MountPoint {
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
				var freeInodes uint64 = 0
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
				} else {
					log.Printf("Bad total bytes: %v", totalBytes)
				}

				totalInodes = statfs.Files
				freeInodes = statfs.Ffree
				if totalInodes > 0 {
					inodesFreePercent = float64(freeInodes) / float64(totalInodes)
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
					InodesInUse:          totalInodes - freeInodes,
					FreeInodesPercentage: inodesFreePercent,
				}

				diskUsageData[mnt.MountPoint] = usage
			}
		}
	}

	return diskUsageData
}

const DiskUsageUpdateInterval = 30 * time.Second

type CachedDiskUsage struct {
	LastUsage   map[string]DiskUsage
	lastUpdated *time.Time
}

func (w *CachedDiskUsage) getUpdateInterval() time.Duration {
	return DiskUsageUpdateInterval
}

func (w *CachedDiskUsage) getLastUpdated() *time.Time {
	return w.lastUpdated
}

func (w *CachedDiskUsage) setLastUpdated(t time.Time) {
	w.lastUpdated = &t
}

func (w *CachedDiskUsage) update() {
	if shouldUpdate(w) {
		w.LastUsage = loadDiskUsage()
	}
}

func NewCachedDiskUsage() *CachedDiskUsage {
	// Build it
	w := &CachedDiskUsage{}

	w.update()

	return w
}

var cachedDiskUsage = NewCachedDiskUsage()

////////////////////////////////////////////
// Widget: Disk
////////////////////////////////////////////

const DiskHeaderText = "--- Disks ---"

type DiskColumn struct {
	column  *ui.Row
	header  *ui.Par
	widgets []*ui.Gauge
}

func NewDiskColumn(span int, offset int) *DiskColumn {
	c := ui.NewCol(span, offset)

	h := ui.NewPar(DiskHeaderText)
	h.Border = false
	h.TextFgColor = ui.ColorGreen
	h.Height = 1

	column := &DiskColumn{
		column:  c,
		header:  h,
		widgets: make([]*ui.Gauge, 0),
	}

	column.update()

	return column
}

func (w *DiskColumn) getGridWidget() ui.GridBufferer {
	return w.column
}

func (w *DiskColumn) getColumn() *ui.Row {
	return w.column
}

func (w *DiskColumn) update() {
	w.header.Text = centerString(w.header.Width, DiskHeaderText)
	//w.header.Text = DiskHeaderText

	gauges := make([]*ui.Gauge, 0)

	for _, d := range cachedDiskUsage.LastUsage {
		gauges = append(gauges, NewDiskGauge(d))
	}

	sort.Sort(ByMountPoint(gauges))

	w.column.Cols = []*ui.Row{}
	ir := w.column

	for _, widget := range gauges {
		nr := &ui.Row{Span: 12, Widget: widget}
		ir.Cols = []*ui.Row{nr}
		ir = nr
	}
}

func (w *DiskColumn) resize() {
	// Do nothing
}

func NewDiskGauge(usage DiskUsage) *ui.Gauge {
	free := int(100 * usage.FreePercentage)
	g := ui.NewGauge()
	g.BorderLabel = usage.MountPoint
	g.Height = 3
	g.Percent = free
	g.Label = fmt.Sprintf("Free: %s/%s (%d%%)",
		prettyPrintBytes(usage.AvailableSizeInBytes), prettyPrintBytes(usage.TotalSizeInBytes), free)
	g.PercentColor = ui.ColorWhite | ui.AttrBold

	g.BarColor = percentToAttribute(free, 0, 100, false)

	return g
}

type ByMountPoint []*ui.Gauge

func (a ByMountPoint) Len() int           { return len(a) }
func (a ByMountPoint) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByMountPoint) Less(i, j int) bool { return a[i].BorderLabel < a[j].BorderLabel }
