package qr

type ErrorCorrection string

const (
	ErrorCorrectionLow     ErrorCorrection = "L"
	ErrorCorrectionMedium  ErrorCorrection = "M"
	ErrorCorrectionQuart   ErrorCorrection = "Q"
	ErrorCorrectionHighest ErrorCorrection = "H"
)

type ModuleStyle string

const (
	ModuleSquare  ModuleStyle = "square"
	ModuleRounded ModuleStyle = "rounded"
	ModuleDots    ModuleStyle = "dots"
	ModuleDiamond ModuleStyle = "diamond"
	ModuleClassy  ModuleStyle = "classy"
)

type CornerOuterStyle string

const (
	CornerOuterSquare       CornerOuterStyle = "square"
	CornerOuterRounded      CornerOuterStyle = "rounded"
	CornerOuterExtraRounded CornerOuterStyle = "extra-rounded"
)

type CornerInnerStyle string

const (
	CornerInnerSquare  CornerInnerStyle = "square"
	CornerInnerDot     CornerInnerStyle = "dot"
	CornerInnerRounded CornerInnerStyle = "rounded"
)

type Gradient struct {
	Enabled bool
	Start   string
	End     string
}

type Style struct {
	Foreground string
	Background string

	Gradient Gradient

	Module      ModuleStyle
	CornerOuter CornerOuterStyle
	CornerInner CornerInnerStyle

	HasLogo bool
}

func DefaultStyle() Style {
	return Style{
		Foreground: "#000000",
		Background: "#ffffff",

		Module:      ModuleSquare,
		CornerOuter: CornerOuterSquare,
		CornerInner: CornerInnerSquare,
	}
}

func (s Style) ErrorCorrection() ErrorCorrection {
	switch {
	case s.HasLogo:
		return ErrorCorrectionHighest

	case s.Gradient.Enabled:
		return ErrorCorrectionQuart

	case s.Module != ModuleSquare ||
		s.CornerOuter != CornerOuterSquare ||
		s.CornerInner != CornerInnerSquare:
		return ErrorCorrectionMedium

	default:
		return ErrorCorrectionLow
	}
}
