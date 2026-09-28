package web

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jayzone91/media-converter/internal/converter"
)

type pdfCompressionResult struct {
	Path string

	OriginalSize int64
	ResultSize   int64

	Unchanged bool
}

func (s *Server) ensurePDFCompressionResult(
	ctx context.Context,
	upload storedPDFUpload,
	mode string,
) (pdfCompressionResult, error) {
	if !validPDFCompressionMode(
		mode,
	) {
		return pdfCompressionResult{},
			fmt.Errorf(
				"unsupported PDF compression mode: %s",
				mode,
			)
	}

	if result, ok :=
		readCachedPDFCompressionResult(
			upload,
			mode,
		); ok {
		return result, nil
	}

	key :=
		upload.ID +
			"\x00" +
			mode

	return s.pdfCompressionFlights.Do(
		ctx,
		key,
		func() (
			pdfCompressionResult,
			error,
		) {
			return s.createPDFCompressionResult(
				ctx,
				upload,
				mode,
			)
		},
	)
}

func (s *Server) createPDFCompressionResult(
	ctx context.Context,
	upload storedPDFUpload,
	mode string,
) (pdfCompressionResult, error) {
	if result, ok :=
		readCachedPDFCompressionResult(
			upload,
			mode,
		); ok {
		return result, nil
	}

	if err :=
		s.acquireConversionSlot(
			ctx,
		); err != nil {
		return pdfCompressionResult{},
			fmt.Errorf(
				"failed to acquire compression slot: %w",
				err,
			)
	}

	defer s.releaseConversionSlot()

	if result, ok :=
		readCachedPDFCompressionResult(
			upload,
			mode,
		); ok {
		return result, nil
	}

	cacheDirectory :=
		pdfCompressionCacheDirectory(
			upload,
		)

	if err :=
		os.MkdirAll(
			cacheDirectory,
			0700,
		); err != nil {
		return pdfCompressionResult{},
			fmt.Errorf(
				"failed to create PDF compression cache directory: %w",
				err,
			)
	}

	tempFile, err :=
		os.CreateTemp(
			cacheDirectory,
			mode+"-*.pdf",
		)

	if err != nil {
		return pdfCompressionResult{},
			fmt.Errorf(
				"failed to create PDF compression temporary file: %w",
				err,
			)
	}

	tempPath :=
		tempFile.Name()

	if err :=
		tempFile.Close(); err != nil {
		_ = os.Remove(
			tempPath,
		)

		return pdfCompressionResult{},
			fmt.Errorf(
				"failed to close PDF compression temporary file: %w",
				err,
			)
	}

	defer os.Remove(
		tempPath,
	)

	if err :=
		s.compressPDF(
			ctx,
			upload.Path,
			tempPath,
			mode,
		); err != nil {
		return pdfCompressionResult{},
			err
	}

	if err :=
		s.validatePDFCompressionOutput(
			ctx,
			tempPath,
			upload.PageCount,
		); err != nil {
		return pdfCompressionResult{},
			err
	}

	info, err :=
		os.Stat(
			tempPath,
		)

	if err != nil {
		return pdfCompressionResult{},
			fmt.Errorf(
				"failed to inspect compressed PDF: %w",
				err,
			)
	}

	if info.Size() <= 0 {
		return pdfCompressionResult{},
			fmt.Errorf(
				"compressed PDF is empty",
			)
	}

	if info.Size() >=
		upload.Size {
		if err :=
			writePDFCompressionUnchangedMarker(
				upload,
				mode,
			); err != nil {
			return pdfCompressionResult{},
				err
		}

		return pdfCompressionResult{
			Path: upload.Path,

			OriginalSize: upload.Size,

			ResultSize: upload.Size,

			Unchanged: true,
		}, nil
	}

	cachePath :=
		pdfCompressionCachePath(
			upload,
			mode,
		)

	if err :=
		os.Rename(
			tempPath,
			cachePath,
		); err != nil {
		if existing, statErr :=
			os.Stat(
				cachePath,
			); statErr == nil &&
			existing.Size() > 0 {
			return pdfCompressionResult{
				Path: cachePath,

				OriginalSize: upload.Size,

				ResultSize: existing.Size(),
			}, nil
		}

		return pdfCompressionResult{},
			fmt.Errorf(
				"failed to cache compressed PDF: %w",
				err,
			)
	}

	return pdfCompressionResult{
		Path: cachePath,

		OriginalSize: upload.Size,

		ResultSize: info.Size(),
	}, nil
}

func readCachedPDFCompressionResult(
	upload storedPDFUpload,
	mode string,
) (pdfCompressionResult, bool) {
	cachePath :=
		pdfCompressionCachePath(
			upload,
			mode,
		)

	info, err :=
		os.Stat(
			cachePath,
		)

	if err == nil &&
		info.Size() > 0 {
		return pdfCompressionResult{
			Path: cachePath,

			OriginalSize: upload.Size,

			ResultSize: info.Size(),
		}, true
	}

	markerPath :=
		pdfCompressionUnchangedMarkerPath(
			upload,
			mode,
		)

	if _, err :=
		os.Stat(
			markerPath,
		); err == nil {
		return pdfCompressionResult{
			Path: upload.Path,

			OriginalSize: upload.Size,

			ResultSize: upload.Size,

			Unchanged: true,
		}, true
	}

	return pdfCompressionResult{},
		false
}

func writePDFCompressionUnchangedMarker(
	upload storedPDFUpload,
	mode string,
) error {
	path :=
		pdfCompressionUnchangedMarkerPath(
			upload,
			mode,
		)

	if err :=
		os.WriteFile(
			path,
			[]byte(
				"unchanged",
			),
			0600,
		); err != nil {
		return fmt.Errorf(
			"failed to cache unchanged PDF compression result: %w",
			err,
		)
	}

	return nil
}

func pdfCompressionCacheDirectory(
	upload storedPDFUpload,
) string {
	return filepath.Join(
		upload.Directory,
		"compression",
	)
}

func pdfCompressionCachePath(
	upload storedPDFUpload,
	mode string,
) string {
	return filepath.Join(
		pdfCompressionCacheDirectory(
			upload,
		),
		mode+".pdf",
	)
}

func pdfCompressionUnchangedMarkerPath(
	upload storedPDFUpload,
	mode string,
) string {
	return filepath.Join(
		pdfCompressionCacheDirectory(
			upload,
		),
		mode+".unchanged",
	)
}

func (s *Server) compressPDF(
	ctx context.Context,
	input string,
	output string,
	mode string,
) error {
	switch mode {
	case "lossless":
		return s.qpdf.Optimize(
			ctx,
			input,
			output,
		)

	case "balanced":
		return s.ghostscript.CompressPDF(
			ctx,
			input,
			output,
			converter.PDFCompressionBalanced,
		)

	case "strong":
		return s.ghostscript.CompressPDF(
			ctx,
			input,
			output,
			converter.PDFCompressionStrong,
		)

	default:
		return fmt.Errorf(
			"unsupported PDF compression mode: %s",
			mode,
		)
	}
}

func validPDFCompressionMode(
	mode string,
) bool {
	switch mode {
	case
		"lossless",
		"balanced",
		"strong":

		return true

	default:
		return false
	}
}

func compressionWasCancelled(
	err error,
	ctx context.Context,
) bool {
	return errors.Is(
		err,
		context.Canceled,
	) ||
		errors.Is(
			err,
			context.DeadlineExceeded,
		) ||
		ctx.Err() != nil
}
