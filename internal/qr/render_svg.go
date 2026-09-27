package qr

import (
	"fmt"
	"strings"
)

const quietZoneModules = 4

func RenderSVG(
	matrix Matrix,
	style Style,
) ([]byte, error) {
	if err := validateSVGInput(
		matrix,
		style,
	); err != nil {
		return nil, err
	}

	totalSize := matrix.Size +
		quietZoneModules*2

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

	renderSVGLogo(
		&svg,
		matrix.Size,
		style,
	)

	svg.WriteString(
		`</svg>`,
	)

	return []byte(
		svg.String(),
	), nil
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

func svgForeground(
	style Style,
) string {
	if style.Gradient.Enabled {
		return "url(#qr-gradient)"
	}

	return style.Foreground
}
