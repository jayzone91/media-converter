package qr

import (
	"fmt"
	"strings"
)

func renderSVGFinderPatterns(
	svg *strings.Builder,
	size int,
	style Style,
) {
	positions := [][2]int{
		{
			quietZoneModules,
			quietZoneModules,
		},
		{
			quietZoneModules + size - 7,
			quietZoneModules,
		},
		{
			quietZoneModules,
			quietZoneModules + size - 7,
		},
	}

	for _, position := range positions {
		renderSVGFinder(
			svg,
			position[0],
			position[1],
			style,
		)
	}
}

func renderSVGFinder(
	svg *strings.Builder,
	x int,
	y int,
	style Style,
) {
	fill := svgForeground(
		style,
	)

	renderSVGFinderOuter(
		svg,
		x,
		y,
		style.CornerOuter,
		fill,
		style.Background,
	)

	renderSVGFinderInner(
		svg,
		x+2,
		y+2,
		style.CornerInner,
		fill,
	)
}

func renderSVGFinderOuter(
	svg *strings.Builder,
	x int,
	y int,
	style CornerOuterStyle,
	fill string,
	background string,
) {
	var radius float64

	switch style {
	case CornerOuterRounded:
		radius = 1.0

	case CornerOuterExtraRounded:
		radius = 1.8
	}

	fmt.Fprintf(
		svg,
		`<rect x="%d" y="%d" width="7" height="7" rx="%.2f" fill="%s"/>`,
		x,
		y,
		radius,
		fill,
	)

	innerRadius := radius * 0.55

	fmt.Fprintf(
		svg,
		`<rect x="%d" y="%d" width="5" height="5" rx="%.2f" fill="%s"/>`,
		x+1,
		y+1,
		innerRadius,
		background,
	)
}

func renderSVGFinderInner(
	svg *strings.Builder,
	x int,
	y int,
	style CornerInnerStyle,
	fill string,
) {
	switch style {
	case CornerInnerDot:
		fmt.Fprintf(
			svg,
			`<circle cx="%.2f" cy="%.2f" r="1.5" fill="%s"/>`,
			float64(x)+1.5,
			float64(y)+1.5,
			fill,
		)

	case CornerInnerRounded:
		fmt.Fprintf(
			svg,
			`<rect x="%d" y="%d" width="3" height="3" rx="0.7" fill="%s"/>`,
			x,
			y,
			fill,
		)

	default:
		fmt.Fprintf(
			svg,
			`<rect x="%d" y="%d" width="3" height="3" fill="%s"/>`,
			x,
			y,
			fill,
		)
	}
}

func isFinderModule(
	x int,
	y int,
	size int,
) bool {
	if x >= 0 &&
		x < 7 &&
		y >= 0 &&
		y < 7 {
		return true
	}

	if x >= size-7 &&
		x < size &&
		y >= 0 &&
		y < 7 {
		return true
	}

	if x >= 0 &&
		x < 7 &&
		y >= size-7 &&
		y < size {
		return true
	}

	return false
}
