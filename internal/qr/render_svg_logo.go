package qr

import (
	"fmt"
	"strings"
)

func renderSVGLogo(
	svg *strings.Builder,
	matrixSize int,
	style Style,
) {
	if !style.HasLogo ||
		style.Logo == "" {
		return
	}

	totalSize := matrixSize +
		quietZoneModules*2

	logoSize := float64(
		totalSize,
	) * 0.20

	padding := logoSize * 0.12

	backgroundSize := logoSize +
		padding*2

	center := float64(
		totalSize,
	) / 2

	backgroundX := center -
		backgroundSize/2

	backgroundY := center -
		backgroundSize/2

	logoX := center -
		logoSize/2

	logoY := center -
		logoSize/2

	fmt.Fprintf(
		svg,
		`<rect x="%.3f" y="%.3f" width="%.3f" height="%.3f" rx="%.3f" fill="%s"/>`,
		backgroundX,
		backgroundY,
		backgroundSize,
		backgroundSize,
		backgroundSize*0.12,
		style.Background,
	)

	fmt.Fprintf(
		svg,
		`<image href="%s" x="%.3f" y="%.3f" width="%.3f" height="%.3f" preserveAspectRatio="xMidYMid meet"/>`,
		style.Logo,
		logoX,
		logoY,
		logoSize,
		logoSize,
	)
}
