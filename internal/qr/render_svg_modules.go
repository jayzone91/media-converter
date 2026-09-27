package qr

import (
	"fmt"
	"strings"
)

func renderSVGModules(
	svg *strings.Builder,
	matrix Matrix,
	style Style,
) {
	fill := svgForeground(
		style,
	)

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
		renderSVGRoundedModule(
			svg,
			x,
			y,
			fill,
		)

	case ModuleDots:
		renderSVGDotModule(
			svg,
			x,
			y,
			fill,
		)

	case ModuleDiamond:
		renderSVGDiamondModule(
			svg,
			x,
			y,
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

func renderSVGRoundedModule(
	svg *strings.Builder,
	x int,
	y int,
	fill string,
) {
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
}

func renderSVGDotModule(
	svg *strings.Builder,
	x int,
	y int,
	fill string,
) {
	fmt.Fprintf(
		svg,
		`<circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s"/>`,
		float64(x)+0.5,
		float64(y)+0.5,
		0.43,
		fill,
	)
}

func renderSVGDiamondModule(
	svg *strings.Builder,
	x int,
	y int,
	fill string,
) {
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
