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
 *
 * TODO: Figure out why ui.ColorRGB doesn't work
 */
func Color8Bit(index int) ui.Attribute {
	retval := ui.ColorBlack

	if index < 8 {
		// Dim colors
	} else if index < 16 {
		// Bright colors
	} else if index < 232 {
		// Palletized colors
		i := index - 16
		r := i / 36
		i -= r * 36
		g := i / 6
		i -= g * 6
		b := i

		smallColor := ui.ColorBlack

		if r >= 3 {
			// Red on
			if g >= 3 {
				// Green on
				if b >= 3 {
					// Blue on
					smallColor = ui.ColorWhite + ui.AttrBold
				} else {
					// Blue off
					smallColor = ui.ColorYellow + ui.AttrBold
				}
			} else {
				// Green off
				if b >= 3 {
					// Blue on
					smallColor = ui.ColorMagenta + ui.AttrBold
				} else {
					// Blue off
					smallColor = ui.ColorRed + ui.AttrBold
				}
			}
		} else {
			// Red off
			if g >= 3 {
				// Green on
				if b >= 3 {
					// Blue on
					smallColor = ui.ColorCyan + ui.AttrBold
				} else {
					// Blue off
					smallColor = ui.ColorGreen + ui.AttrBold
				}
			} else {
				// Green off
				if b >= 3 {
					// Blue on
					smallColor = ui.ColorBlue + ui.AttrBold
				} else {
					// Blue off
					smallColor = ui.ColorBlack
				}
			}
		}

		retval = smallColor
	} else {
		// Grayscale colors
		if index < 238 {
			retval = ui.ColorBlack
		} else if index < 244 {
			retval = ui.ColorWhite
		} else if index < 250 {
			retval = ui.ColorBlack + ui.AttrBold
		} else if index < 256 {
			retval = ui.ColorWhite + ui.AttrBold
		}
	}

	return retval

}

/**
 * Converts 8-bit color into 3/4-bit color.
 * https://en.wikipedia.org/wiki/ANSI_escape_code#8-bit
 *
 * TODO: Figure out why ui.ColorRGB doesn't work
 */
func Color8BitAsString(index int) string {
	retval := "fg-black"

	if index < 8 {
		// Dim colors
	} else if index < 16 {
		// Bright colors
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
