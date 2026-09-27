package qr

import (
	"fmt"

	qrcode "github.com/yeqown/go-qrcode/v2"
)

type Matrix struct {
	Bitmap [][]bool

	Size    int
	Version int

	ErrorCorrection ErrorCorrection
}

func Generate(
	payload string,
	style Style,
) (Matrix, error) {
	if payload == "" {
		return Matrix{}, fmt.Errorf(
			"QR payload darf nicht leer sein",
		)
	}

	errorCorrection :=
		style.ErrorCorrection()

	option := errorCorrectionOption(
		errorCorrection,
	)

	code, err := qrcode.NewWith(
		payload,
		option,
	)
	if err != nil {
		return Matrix{}, fmt.Errorf(
			"QR Code konnte nicht erzeugt werden: %w",
			err,
		)
	}

	writer := &matrixWriter{}

	if err := code.Save(writer); err != nil {
		return Matrix{}, fmt.Errorf(
			"QR Matrix konnte nicht gelesen werden: %w",
			err,
		)
	}

	if len(writer.bitmap) == 0 {
		return Matrix{}, fmt.Errorf(
			"QR Matrix ist leer",
		)
	}

	size := len(writer.bitmap)

	return Matrix{
		Bitmap: writer.bitmap,

		Size: size,

		Version: qrVersionFromSize(
			size,
		),

		ErrorCorrection: errorCorrection,
	}, nil
}

func errorCorrectionOption(
	level ErrorCorrection,
) qrcode.EncodeOption {
	switch level {
	case ErrorCorrectionHighest:
		return qrcode.WithErrorCorrectionLevel(
			qrcode.ErrorCorrectionHighest,
		)

	case ErrorCorrectionQuart:
		return qrcode.WithErrorCorrectionLevel(
			qrcode.ErrorCorrectionQuart,
		)

	case ErrorCorrectionMedium:
		return qrcode.WithErrorCorrectionLevel(
			qrcode.ErrorCorrectionMedium,
		)

	default:
		return qrcode.WithErrorCorrectionLevel(
			qrcode.ErrorCorrectionLow,
		)
	}
}

func qrVersionFromSize(
	size int,
) int {
	if size < 21 {
		return 0
	}

	return (size - 17) / 4
}

type matrixWriter struct {
	bitmap [][]bool
}

func (w *matrixWriter) Write(
	matrix qrcode.Matrix,
) error {
	source := matrix.Bitmap()

	w.bitmap = make(
		[][]bool,
		len(source),
	)

	for y, row := range source {
		w.bitmap[y] = append(
			[]bool(nil),
			row...,
		)
	}

	return nil
}

func (w *matrixWriter) Close() error {
	return nil
}
