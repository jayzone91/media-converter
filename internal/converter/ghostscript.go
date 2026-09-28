package converter

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type Ghostscript struct {
	binary string
}

type PDFCompressionPreset string

const (
	PDFCompressionBalanced PDFCompressionPreset = "balanced"
	PDFCompressionStrong   PDFCompressionPreset = "strong"
)

func NewGhostscript() (*Ghostscript, error) {
	candidates := []string{
		"gs",
	}

	if runtime.GOOS == "windows" {
		candidates = []string{
			"gswin64c",
			"gswin32c",
			"gs",
		}
	}

	for _, candidate := range candidates {
		binary, err := exec.LookPath(candidate)
		if err == nil {
			return &Ghostscript{
				binary: binary,
			}, nil
		}
	}

	return nil, fmt.Errorf(
		"ghostscript not found",
	)
}

func (g *Ghostscript) CompressPDF(
	ctx context.Context,
	input string,
	output string,
	preset PDFCompressionPreset,
) error {
	args, err := ghostscriptCompressionArgs(
		input,
		output,
		preset,
	)
	if err != nil {
		return err
	}

	cmd := externalCommandContext(
		ctx,
		g.binary,
		args...,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf(
				"ghostscript compression failed: %w",
				ctxErr,
			)
		}

		message := strings.TrimSpace(
			stderr.String(),
		)

		if message == "" {
			message = strings.TrimSpace(
				stdout.String(),
			)
		}

		return fmt.Errorf(
			"ghostscript compression failed: %w: %s",
			err,
			message,
		)
	}

	return nil
}

func ghostscriptCompressionArgs(
	input string,
	output string,
	preset PDFCompressionPreset,
) ([]string, error) {
	args := []string{
		"-dSAFER",
		"-dBATCH",
		"-dNOPAUSE",
		"-dQUIET",
		"-sDEVICE=pdfwrite",
		"-dCompatibilityLevel=1.7",
		"-dDetectDuplicateImages=true",
		"-dCompressFonts=true",
		"-dSubsetFonts=true",
		"-dAutoRotatePages=/None",
	}

	switch preset {
	case PDFCompressionBalanced:
		args = append(
			args,
			"-dDownsampleColorImages=true",
			"-dColorImageDownsampleType=/Bicubic",
			"-dColorImageResolution=150",
			"-dEncodeColorImages=true",
			"-dColorImageFilter=/DCTEncode",
			"-dJPEGQ=82",

			"-dDownsampleGrayImages=true",
			"-dGrayImageDownsampleType=/Bicubic",
			"-dGrayImageResolution=150",
			"-dEncodeGrayImages=true",
			"-dGrayImageFilter=/DCTEncode",

			"-dDownsampleMonoImages=true",
			"-dMonoImageDownsampleType=/Subsample",
			"-dMonoImageResolution=300",
		)

	case PDFCompressionStrong:
		args = append(
			args,
			"-dDownsampleColorImages=true",
			"-dColorImageDownsampleType=/Bicubic",
			"-dColorImageResolution=96",
			"-dEncodeColorImages=true",
			"-dColorImageFilter=/DCTEncode",
			"-dJPEGQ=65",

			"-dDownsampleGrayImages=true",
			"-dGrayImageDownsampleType=/Bicubic",
			"-dGrayImageResolution=96",
			"-dEncodeGrayImages=true",
			"-dGrayImageFilter=/DCTEncode",

			"-dDownsampleMonoImages=true",
			"-dMonoImageDownsampleType=/Subsample",
			"-dMonoImageResolution=200",
		)

	default:
		return nil, fmt.Errorf(
			"unsupported PDF compression preset: %s",
			preset,
		)
	}

	args = append(
		args,
		"-sOutputFile="+output,
		input,
	)

	return args, nil
}
