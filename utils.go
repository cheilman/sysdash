package main

/**
 * Utility methods/classes for sysdash
 */

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"unicode/utf8"

	ui "github.com/gizak/termui"
)

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

	var rightJustify = ""
	if rightJustfyLen > 0 {
		rightJustify = strings.Repeat(" ", rightJustfyLen)
	}

	return rightJustify + str
}

func centerString(width int, str string) string {
	start := (width / 2) - (utf8.RuneCountInString(str) / 2)

	if start > 0 {
		return fmt.Sprintf("%s%s", strings.Repeat(" ", start), str)
	} else {
		return str
	}
}

var ANSI_REGEXP = regexp.MustCompile(`\x1B\[(([0-9]{1,2})?(;)?([0-9]{1,2})?)?[m,K,H,f,J]`)

func stripANSI(str string) string {
	return ANSI_REGEXP.ReplaceAllLiteralString(str, "")
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

var FG_BG_REGEXP = regexp.MustCompile("(fg|bg|FG|BG)-")

// Colors according to where value is in the min/max range
func percentToAttribute(value int, minValue int, maxValue int, invert bool) ui.Attribute {
	return ui.StringToAttribute(FG_BG_REGEXP.ReplaceAllLiteralString(percentToAttributeString(value, minValue, maxValue, invert), ""))
}

// Colors according to where value is in the min/max range
func percentToAttributeString(value int, minValue int, maxValue int, invert bool) string {
	span := float64(maxValue - minValue)
	fvalue := float64(value)

	// If invert is set...
	if invert {
		// "good" is close to min and "bad" is closer to max
		if fvalue > 0.90*span {
			return "fg-red,fg-bold"
		} else if fvalue > 0.75*span {
			return "fg-red"
		} else if fvalue > 0.50*span {
			return "fg-yellow,fg-bold"
		} else if fvalue > 0.25*span {
			return "fg-green"
		} else if fvalue > 0.05*span {
			return "fg-green,fg-bold"
		} else {
			return "fg-blue,fg-bold"
		}
	} else {
		// "good" is close to max and "bad" is closer to min
		if fvalue < 0.10*span {
			return "fg-red,fg-bold"
		} else if fvalue < 0.25*span {
			return "fg-red"
		} else if fvalue < 0.50*span {
			return "fg-yellow,fg-bold"
		} else if fvalue < 0.75*span {
			return "fg-green"
		} else if fvalue < 0.95*span {
			return "fg-green,fg-bold"
		} else {
			return "fg-blue,fg-bold"
		}
	}
}

////////////////////////////////////////////
// Utility: Command Exec
////////////////////////////////////////////

func execAndGetOutput(name string, workingDirectory *string, args ...string) (stdout string, exitCode int, err error) {
	cmd := exec.Command(name, args...)

	var out bytes.Buffer
	cmd.Stdout = &out

	if workingDirectory != nil {
		cmd.Dir = *workingDirectory
	}

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
// Utility: Paths
////////////////////////////////////////////

func normalizePath(osPathname string) string {
	// Get absolute path with no symlinks
	nolinksPath, symErr := filepath.EvalSymlinks(osPathname)
	if symErr != nil {
		log.Printf("Error evaluating file symlinks (%v): %v", osPathname, symErr)
		return osPathname
	} else {
		fullName, pathErr := filepath.Abs(nolinksPath)

		if pathErr != nil {
			log.Printf("Error getting absolute path (%v): %v", nolinksPath, pathErr)
			return nolinksPath
		} else {
			return fullName
		}
	}
}

////////////////////////////////////////////
// Utility: 8-bit ANSI Colors
////////////////////////////////////////////

/**
 * Converts 8-bit color into 3/4-bit color.
 * https://en.wikipedia.org/wiki/ANSI_escape_code#8-bit
 */
func Color8BitAsString(index int) string {
	retval := "fg-black"

	if index < 16 {
		switch index {
		case 0:
			retval = "fg-black"
		case 1:
			retval = "fg-red"
		case 2:
			retval = "fg-green"
		case 3:
			retval = "fg-yellow"
		case 4:
			retval = "fg-blue"
		case 5:
			retval = "fg-magenta"
		case 6:
			retval = "fg-cyan"
		case 7:
			retval = "fg-white"
		case 8:
			retval = "fg-black,fg-bold"
		case 9:
			retval = "fg-red,fg-bold"
		case 10:
			retval = "fg-green,fg-bold"
		case 11:
			retval = "fg-yellow,fg-bold"
		case 12:
			retval = "fg-blue,fg-bold"
		case 13:
			retval = "fg-magenta,fg-bold"
		case 14:
			retval = "fg-cyan,fg-bold"
		case 15:
			retval = "fg-white,fg-bold"
		}
	} else if index < 232 {
		// Palletized colors
		i := index - 16
		r := i / 36
		i -= r * 36
		g := i / 6
		i -= g * 6
		b := i

		smallColor := "fg-black"

		if r >= 3 {
			// Red on
			if g >= 3 {
				// Green on
				if b >= 3 {
					// Blue on
					smallColor = "fg-white,fg-bold"
				} else {
					// Blue off
					smallColor = "fg-yellow,fg-bold"
				}
			} else {
				// Green off
				if b >= 3 {
					// Blue on
					smallColor = "fg-magenta,fg-bold"
				} else {
					// Blue off
					smallColor = "fg-red,fg-bold"
				}
			}
		} else {
			// Red off
			if g >= 3 {
				// Green on
				if b >= 3 {
					// Blue on
					smallColor = "fg-cyan,fg-bold"
				} else {
					// Blue off
					smallColor = "fg-green,fg-bold"
				}
			} else {
				// Green off
				if b >= 3 {
					// Blue on
					smallColor = "fg-blue,fg-bold"
				} else {
					// Blue off
					smallColor = "fg-black"
				}
			}
		}

		retval = smallColor
	} else {
		// Grayscale colors
		if index < 238 {
			retval = "fg-black"
		} else if index < 244 {
			retval = "fg-white"
		} else if index < 250 {
			retval = "fg-black,fg-bold"
		} else if index < 256 {
			retval = "fg-white,fg-bold"
		}
	}

	return retval

}

//////////////////////////////////////////////
// Utility: Convert ANSI to (fg-color) syntax
//////////////////////////////////////////////

var ANSI_COLOR_GROUPING_REGEXP = regexp.MustCompile(`\x1B\x5B(?P<sgr>(?:[0-9]+;?)+)m(?P<content>[^\x1B]+)\x1B\x5B0?m`)

var ANSI_COLOR_MAPPINGS = map[int]string{
	1:  "fg-bold",
	30: "fg-black",
	31: "fg-red",
	32: "fg-green",
	33: "fg-yellow",
	34: "fg-blue",
	35: "fg-magenta",
	36: "fg-cyan",
	37: "fg-white",
	40: "fg-black",
	41: "fg-red",
	42: "fg-green",
	43: "fg-yellow",
	44: "fg-blue",
	45: "fg-magenta",
	46: "fg-cyan",
	47: "fg-white",
}

func palletizedColorToString(index int) string {
	return Color8BitAsString(index)
}

func rgbColorToString(r int, g int, b int) string {
	log.Printf("We don't know how to handle RGB color yet.  Color: #%02x%02x%02x)", r, g, b)
	return "fg-white"
}

// Returns how many elements were consumed and the color string
func SGR256ColorToString(parts []int) (int, string) {
	if len(parts) < 1 {
		log.Printf("Error parsing 256-color SGR code (bad length).  Length: %d, Parts: %v", len(parts), parts)
		return 1, "fg-white"
	}

	switch parts[0] {
	case 2:
		if len(parts) < 4 {
			log.Printf("Error parsing 256-color SGR code (not enough numbers for RGB).  Parts: %v", parts)
			return 1, "fg-white"
		} else {
			return 4, rgbColorToString(parts[1], parts[2], parts[3])
		}
	case 5:
		if len(parts) < 2 {
			log.Printf("Error parsing 256-color SGR code (no index for palette).  Parts: %v", parts)
			return 1, "fg-white"
		} else {
			return 2, palletizedColorToString(parts[1])
		}
	default:
		log.Printf("Error parsing 256-color SGR code (bad code).  Code: %d, Parts: %v", parts[0], parts)
		return 1, "fg-white"
	}
}

func SGRToColorString(sgr string) string {
	parts := strings.Split(sgr, ";")
	iparts := make([]int, len(parts))

	for i, x := range parts {
		iparts[i], _ = strconv.Atoi(x)
	}

	i := 0
	retval := ""

	appendRet := func(str string) {
		if len(retval) > 0 {
			retval += "," + str
		} else {
			retval += str
		}
	}

	for i < len(iparts) {
		if val, ok := ANSI_COLOR_MAPPINGS[iparts[i]]; ok {
			// if it's in the map, use that
			appendRet(val)
		} else {
			switch iparts[i] {
			case 38:
				// Foreground palette or RGB
				relevantSlice := iparts[i+1:]
				consumed, color := SGR256ColorToString(relevantSlice)

				i += consumed
				appendRet(color)

			case 48:
				// Background palette or RGB
				relevantSlice := iparts[i+1:]
				consumed, color := SGR256ColorToString(relevantSlice)

				color = strings.Replace(color, "fg", "bg", -1)

				i += consumed
				appendRet(color)

			}
		}

		i++
	}

	return retval
}

func ConvertANSIToColorStrings(ansi string) string {
	log.Printf("Looking for matches in '%v'", ansi)
	retval := ANSI_COLOR_GROUPING_REGEXP.ReplaceAllStringFunc(ansi, func(matchStr string) string {
		// matchStr should be the regexp, let's match it again to get the groupings
		matches := ANSI_COLOR_GROUPING_REGEXP.FindStringSubmatch(matchStr)

		// 0 is the whole string, 1+ are match groups
		sgr := matches[1]
		content := matches[2]

		colorStr := SGRToColorString(sgr)
		coloredContent := fmt.Sprintf("[%v](%v)", content, colorStr)

		return coloredContent
	})

	return stripANSI(retval)
}
