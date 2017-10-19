package main

/**
 * Networking Info
 */

import (
	"fmt"
	"log"
	"net"

	ui "github.com/gizak/termui"
)

////////////////////////////////////////////
// Widget: Network
////////////////////////////////////////////

type NetworkWidget struct {
	widget *ui.List
}

func NewNetworkWidget() *NetworkWidget {
	// Create base element
	e := ui.NewList()
	e.Height = 3
	e.Border = true
	e.BorderLabel = "Network"

	// Create widget
	w := &NetworkWidget{
		widget: e,
	}

	w.update()
	w.resize()

	return w
}

func (w *NetworkWidget) getGridWidget() ui.GridBufferer {
	return w.widget
}

func (w *NetworkWidget) update() {
	w.widget.Items = []string{}
	w.widget.Height = 2

	// Getting addresses pulled from: https://stackoverflow.com/a/23558495/147354
	ifaces, ifacesErr := net.Interfaces()

	if ifacesErr != nil {
		log.Printf("Error loading network interfaces: %v", ifacesErr)
	} else {
		for _, i := range ifaces {
                        if i.Name == "lo" {
                            continue
                        }

			addrs, addrsErr := i.Addrs()

			if addrsErr != nil {
				log.Printf("Failed to load addresses for interface '%v': %v", i, addrsErr)
			} else {
				for _, addr := range addrs {
					var ip net.IP

					switch v := addr.(type) {
					case *net.IPNet:
						ip = v.IP
					case *net.IPAddr:
						ip = v.IP
					}

					line := fmt.Sprintf("[%10v](fg-cyan): [%15v](fg-blue,fg-bold)", i.Name, ip.String())

					w.widget.Items = append(w.widget.Items, line)
					w.widget.Height += 1
				}
			}
		}

		// TODO: Add WLAN Addresses, Network Location (geoip?)
	}
}

func (w *NetworkWidget) resize() {
	// Do nothing
}
