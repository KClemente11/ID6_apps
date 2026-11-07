package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type panel struct{ width, height int }

var (
	resolution  = regexp.MustCompile(`(?im)^[^\r\n]*resolution:\s*x\s*=\s*([0-9]{2,4}),\s*y\s*=\s*([0-9]{2,4})\b`)
	touchCoding = regexp.MustCompile(`(?im)^#Coded is:[^\r\n]*\btouch_screen\b`)
	position    = regexp.MustCompile(`<property id="(9|10|20|21)" value="([0-9]+)"`)
)

func readPanel(output []byte) panel {
	match := resolution.FindSubmatch(output)
	if match == nil {
		return panel{}
	}
	width, _ := strconv.Atoi(string(match[1]))
	height, _ := strconv.Atoi(string(match[2]))
	if width == 0 || height == 0 {
		return panel{}
	}
	return panel{width, height}
}

func (p panel) String() string {
	if p.width == 0 || p.height == 0 {
		return "Unknown"
	}
	return fmt.Sprintf("%d × %d", p.width, p.height)
}

func (p panel) automatic() bool {
	return p == (panel{1280, 480}) || p == (panel{1440, 540}) || p == (panel{800, 480})
}

func (p panel) supported() bool {
	return p.width >= 800 && p.width <= 1920 && p.height >= 480 && p.height <= 720 &&
		p.width*3 >= p.height*5 && p.width <= p.height*3
}

func manualPanel(value string) panel {
	w, h, _ := strings.Cut(value, "x")
	width, widthErr := strconv.Atoi(w)
	height, heightErr := strconv.Atoi(h)
	if widthErr != nil || heightErr != nil {
		return panel{}
	}
	return panel{width, height}
}

// The bundled XML uses a 1280×480 canvas. Scale only explicit geometry,
// preserving the native menu controls, text sizing, model IDs and actions.
func scaleLayout(xml []byte, name string, p panel) []byte {
	if p == (panel{1280, 480}) {
		return xml
	}
	xScale, yScale := p.width, p.height
	if name == "ShiftDeck" || name == "MhdMaps" {
		// Fit the bitmap canvas inside the panel without stretching it.
		xScale = min(xScale, p.height*1280/480)
		yScale = min(yScale, p.width*480/1280)
	}
	return position.ReplaceAllFunc(xml, func(property []byte) []byte {
		match := position.FindSubmatch(property)
		id := string(match[1])
		value, _ := strconv.Atoi(string(match[2]))
		switch id {
		case "9", "20":
			value = (value*xScale + 640) / 1280
			if id == "20" {
				value += (p.width - xScale) / 2
			}
		case "10", "21":
			value = (value*yScale + 240) / 480
			if id == "21" {
				value += (p.height - yScale) / 2
			}
		}
		return fmt.Appendf(nil, `<property id="%s" value="%d"`, id, value)
	})
}
