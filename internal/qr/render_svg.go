package qr

import (
	"fmt"
	"regexp"
	"strings"
)

const quietZoneModules = 4

var hexColorPattern = regexp.MustCompile(
	`^#[0-9a-fA-F]{6}$`,
)

func RenderSVG(
	matrix Matrix,
	style Style,
) ([]byte, error) {
	if matrix.Size == 0 ||
		len(matrix.Bitmap) != matrix.Size {
		return nil, fmt.Errorf(
			"invalid QR matrix",
		)
	}

	if err := validateStyleColors(style); err != nil {
		return nil, err
	}

	totalSize :=
		matrix.Size +
			(quietZoneModules * 2)

	var svg strings.Builder

	svg.WriteString(
		`<?xml version="1.0" encoding="UTF-8"?>`,
	)

	fmt.Fprintf(
		&svg,
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" shape-rendering="geometricPrecision">`,
		totalSize,
		totalSize,
	)

	renderSVGDefinitions(
		&svg,
		style,
		totalSize,
	)

	fmt.Fprintf(
		&svg,
		`<rect width="%d" height="%d" fill="%s"/>`,
		totalSize,
		totalSize,
		style.Background,
	)

	renderSVGModules(
		&svg,
		matrix,
		style,
	)

	renderSVGFinderPatterns(
		&svg,
		matrix.Size,
		style,
	)

	svg.WriteString(`</svg>`)

	return []byte(svg.String()), nil
}

func renderSVGDefinitions(
	svg *strings.Builder,
	style Style,
	totalSize int,
) {
	if !style.Gradient.Enabled {
		return
	}

	fmt.Fprintf(
		svg,
		`<defs><linearGradient id="qr-gradient" x1="0" y1="0" x2="%d" y2="%d" gradientUnits="userSpaceOnUse"><stop offset="0%%" stop-color="%s"/><stop offset="100%%" stop-color="%s"/></linearGradient></defs>`,
		totalSize,
		totalSize,
		style.Gradient.Start,
		style.Gradient.End,
	)
}

func renderSVGModules(
	svg *strings.Builder,
	matrix Matrix,
	style Style,
) {
	fill := svgForeground(style)

	for y, row := range matrix.Bitmap {
		if len(row) != matrix.Size {
			continue
		}

		for x, dark := range row {
			if !dark {
				continue
			}

			if isFinderModule(
				x,
				y,
				matrix.Size,
			) {
				continue
			}

			renderSVGModule(
				svg,
				x+quietZoneModules,
				y+quietZoneModules,
				style.Module,
				fill,
			)
		}
	}
}

func renderSVGModule(
	svg *strings.Builder,
	x int,
	y int,
	style ModuleStyle,
	fill string,
) {
	switch style {
	case ModuleRounded:
		fmt.Fprintf(
			svg,
			`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="%.2f" fill="%s"/>`,
			float64(x)+0.06,
			float64(y)+0.06,
			0.88,
			0.88,
			0.24,
			fill,
		)

	case ModuleDots:
		fmt.Fprintf(
			svg,
			`<circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s"/>`,
			float64(x)+0.5,
			float64(y)+0.5,
			0.43,
			fill,
		)

	case ModuleDiamond:
		fmt.Fprintf(
			svg,
			`<polygon points="%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f" fill="%s"/>`,
			float64(x)+0.5,
			float64(y)+0.03,
			float64(x)+0.97,
			float64(y)+0.5,
			float64(x)+0.5,
			float64(y)+0.97,
			float64(x)+0.03,
			float64(y)+0.5,
			fill,
		)

	case ModuleClassy:
		renderSVGClassyModule(
			svg,
			float64(x),
			float64(y),
			fill,
		)

	default:
		fmt.Fprintf(
			svg,
			`<rect x="%d" y="%d" width="1" height="1" fill="%s"/>`,
			x,
			y,
			fill,
		)
	}
}

func renderSVGClassyModule(
	svg *strings.Builder,
	x float64,
	y float64,
	fill string,
) {
	fmt.Fprintf(
		svg,
		`<path d="M %.2f %.2f H %.2f Q %.2f %.2f %.2f %.2f V %.2f H %.2f Q %.2f %.2f %.2f %.2f Z" fill="%s"/>`,
		x,
		y,
		x+0.72,
		x+1,
		y,
		x+1,
		y+0.28,
		y+1,
		x+0.28,
		x,
		y+1,
		x,
		y+0.72,
		fill,
	)
}

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
			quietZoneModules +
				size - 7,
			quietZoneModules,
		},
		{
			quietZoneModules,
			quietZoneModules +
				size - 7,
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
	fill := svgForeground(style)

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

	default:
		radius = 0
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

func svgForeground(
	style Style,
) string {
	if style.Gradient.Enabled {
		return "url(#qr-gradient)"
	}

	return style.Foreground
}

func validateStyleColors(
	style Style,
) error {
	if !hexColorPattern.MatchString(
		style.Foreground,
	) {
		return fmt.Errorf(
			"invalid foreground color",
		)
	}

	if !hexColorPattern.MatchString(
		style.Background,
	) {
		return fmt.Errorf(
			"invalid background color",
		)
	}

	if style.Gradient.Enabled {
		if !hexColorPattern.MatchString(
			style.Gradient.Start,
		) {
			return fmt.Errorf(
				"invalid gradient start color",
			)
		}

		if !hexColorPattern.MatchString(
			style.Gradient.End,
		) {
			return fmt.Errorf(
				"invalid gradient end color",
			)
		}
	}

	return nil
}
