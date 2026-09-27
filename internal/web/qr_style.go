package web

import (
	"encoding/base64"
	"fmt"
	"strings"

	qrservice "github.com/jayzone91/media-converter/internal/qr"
)

const maxQRLogoSize = 2 * 1024 * 1024

func buildQRStyle(
	request qrStyleRequest,
) (qrservice.Style, error) {
	style := qrservice.DefaultStyle()

	if request.Foreground != "" {
		style.Foreground = request.Foreground
	}

	if request.Background != "" {
		style.Background = request.Background
	}

	style.Gradient = qrservice.Gradient{
		Enabled: request.GradientEnabled,
		Start:   request.GradientStart,
		End:     request.GradientEnd,
	}

	if err := applyQRModuleStyle(
		&style,
		request.Module,
	); err != nil {
		return style, err
	}

	if err := applyQROuterCornerStyle(
		&style,
		request.CornerOuter,
	); err != nil {
		return style, err
	}

	if err := applyQRInnerCornerStyle(
		&style,
		request.CornerInner,
	); err != nil {
		return style, err
	}

	normalizeQRGradient(
		&style,
	)

	if request.HasLogo {
		logo, err := validateQRLogo(
			request.Logo,
		)
		if err != nil {
			return style, err
		}

		style.HasLogo = true
		style.Logo = logo
	}

	return style, nil
}

func applyQRModuleStyle(
	style *qrservice.Style,
	value string,
) error {
	switch value {
	case "", "square":
		style.Module = qrservice.ModuleSquare

	case "rounded":
		style.Module = qrservice.ModuleRounded

	case "dots":
		style.Module = qrservice.ModuleDots

	case "diamond":
		style.Module = qrservice.ModuleDiamond

	case "classy":
		style.Module = qrservice.ModuleClassy

	default:
		return fmt.Errorf(
			"Ungültiger QR-Code-Stil",
		)
	}

	return nil
}

func applyQROuterCornerStyle(
	style *qrservice.Style,
	value string,
) error {
	switch value {
	case "", "square":
		style.CornerOuter = qrservice.CornerOuterSquare

	case "rounded":
		style.CornerOuter = qrservice.CornerOuterRounded

	case "extra-rounded":
		style.CornerOuter = qrservice.CornerOuterExtraRounded

	default:
		return fmt.Errorf(
			"Ungültiger äußerer Eckenstil",
		)
	}

	return nil
}

func applyQRInnerCornerStyle(
	style *qrservice.Style,
	value string,
) error {
	switch value {
	case "", "square":
		style.CornerInner = qrservice.CornerInnerSquare

	case "dot":
		style.CornerInner = qrservice.CornerInnerDot

	case "rounded":
		style.CornerInner = qrservice.CornerInnerRounded

	default:
		return fmt.Errorf(
			"Ungültiger innerer Eckenstil",
		)
	}

	return nil
}

func normalizeQRGradient(
	style *qrservice.Style,
) {
	if !style.Gradient.Enabled {
		return
	}

	if style.Gradient.Start == "" {
		style.Gradient.Start = style.Foreground
	}

	if style.Gradient.End == "" {
		style.Gradient.End = style.Foreground
	}
}

func validateQRLogo(
	value string,
) (string, error) {
	value = strings.TrimSpace(
		value,
	)

	if value == "" {
		return "", fmt.Errorf(
			"Logo fehlt",
		)
	}

	prefix, err := qrLogoPrefix(
		value,
	)
	if err != nil {
		return "", err
	}

	encoded := strings.TrimPrefix(
		value,
		prefix,
	)

	data, err := base64.StdEncoding.DecodeString(
		encoded,
	)
	if err != nil {
		return "", fmt.Errorf(
			"Logo ist ungültig",
		)
	}

	if len(data) == 0 {
		return "", fmt.Errorf(
			"Logo ist leer",
		)
	}

	if len(data) > maxQRLogoSize {
		return "", fmt.Errorf(
			"Logo darf maximal 2 MiB groß sein",
		)
	}

	return value, nil
}

func qrLogoPrefix(
	value string,
) (string, error) {
	prefixes := []string{
		"data:image/png;base64,",
		"data:image/jpeg;base64,",
		"data:image/webp;base64,",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(
			value,
			prefix,
		) {
			return prefix, nil
		}
	}

	return "", fmt.Errorf(
		"Logo muss PNG, JPEG oder WEBP sein",
	)
}
