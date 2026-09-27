package converter

import (
	"fmt"
	"math"
	"strings"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

const (
	webPDFMarginInches = 0.39

	webPDFCSSPixelsPerInch = 96.0
)

type webPDFRenderProfile struct {
	Mode string

	Width  int64
	Height int64

	DeviceScaleFactor float64

	Mobile bool

	Media string

	AutoScale bool
}

func webPDFRenderProfileFor(
	mode string,
) (webPDFRenderProfile, error) {
	switch strings.ToLower(
		strings.TrimSpace(
			mode,
		),
	) {
	case "", "desktop":
		return webPDFRenderProfile{
			Mode: "desktop",

			Width:  1440,
			Height: 1080,

			DeviceScaleFactor: 1,

			Mobile: false,

			Media: "screen",

			AutoScale: true,
		}, nil

	case "tablet":
		return webPDFRenderProfile{
			Mode: "tablet",

			Width:  1024,
			Height: 1366,

			DeviceScaleFactor: 2,

			Mobile: true,

			Media: "screen",

			AutoScale: true,
		}, nil

	case "mobile":
		return webPDFRenderProfile{
			Mode: "mobile",

			Width:  390,
			Height: 844,

			DeviceScaleFactor: 3,

			Mobile: true,

			Media: "screen",

			AutoScale: true,
		}, nil

	case "print":
		return webPDFRenderProfile{
			Mode: "print",

			Width:  1440,
			Height: 1080,

			DeviceScaleFactor: 1,

			Mobile: false,

			Media: "print",

			AutoScale: false,
		}, nil

	default:
		return webPDFRenderProfile{},
			fmt.Errorf(
				"unsupported webpage render mode: %s",
				mode,
			)
	}
}

func webPDFProfileActions(
	profile webPDFRenderProfile,
) []chromedp.Action {
	return []chromedp.Action{
		emulation.SetDeviceMetricsOverride(
			profile.Width,
			profile.Height,
			profile.DeviceScaleFactor,
			profile.Mobile,
		),

		emulation.
			SetEmulatedMedia().
			WithMedia(
				profile.Media,
			),
	}
}

func webPDFPrintScale(
	profile webPDFRenderProfile,
	paperWidth float64,
	paperHeight float64,
	landscape bool,
) float64 {
	if !profile.AutoScale {
		return 1
	}

	pageWidth :=
		paperWidth

	if landscape {
		pageWidth =
			paperHeight
	}

	printableWidth :=
		pageWidth -
			2*webPDFMarginInches

	if printableWidth <= 0 ||
		profile.Width <= 0 {
		return 1
	}

	printablePixels :=
		printableWidth *
			webPDFCSSPixelsPerInch

	scale :=
		printablePixels /
			float64(
				profile.Width,
			)

	return math.Min(
		2,
		math.Max(
			0.1,
			scale,
		),
	)
}
