package web

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type pdfOptimizationResult struct {
	Path string

	OriginalSize int64
	ResultSize   int64

	Linearized bool
	Unchanged  bool
}

func (s *Server) ensurePDFOptimizationResult(
	ctx context.Context,
	upload storedPDFUpload,
	linearize bool,
) (pdfOptimizationResult, error) {
	if result, ok :=
		readCachedPDFOptimizationResult(
			upload,
			linearize,
		); ok {
		return result, nil
	}

	if err :=
		s.acquireWorkload(
			ctx,
			workloadQPDF,
		); err != nil {
		return pdfOptimizationResult{},
			fmt.Errorf(
				"failed to acquire qpdf optimization workload: %w",
				err,
			)
	}

	defer s.releaseWorkload(
		workloadQPDF,
	)

	if result, ok :=
		readCachedPDFOptimizationResult(
			upload,
			linearize,
		); ok {
		return result, nil
	}

	if err :=
		validatePDFOptimizationInput(
			upload.Path,
		); err != nil {
		return pdfOptimizationResult{},
			err
	}

	cacheDirectory :=
		pdfOptimizationCacheDirectory(
			upload,
		)

	if err :=
		os.MkdirAll(
			cacheDirectory,
			0700,
		); err != nil {
		return pdfOptimizationResult{},
			fmt.Errorf(
				"create optimization cache directory: %w",
				err,
			)
	}

	tempFile, err :=
		os.CreateTemp(
			cacheDirectory,
			"optimize-*.pdf",
		)

	if err != nil {
		return pdfOptimizationResult{},
			fmt.Errorf(
				"create optimization temporary file: %w",
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

		return pdfOptimizationResult{},
			fmt.Errorf(
				"close optimization temporary file: %w",
				err,
			)
	}

	defer os.Remove(
		tempPath,
	)

	if err :=
		s.optimizePDF(
			ctx,
			upload.Path,
			tempPath,
			linearize,
		); err != nil {
		return pdfOptimizationResult{},
			err
	}

	if err :=
		s.validatePDFOptimizationOutput(
			ctx,
			tempPath,
			upload.PageCount,
			linearize,
		); err != nil {
		return pdfOptimizationResult{},
			err
	}

	info, err :=
		os.Stat(
			tempPath,
		)

	if err != nil {
		return pdfOptimizationResult{},
			fmt.Errorf(
				"inspect optimized PDF: %w",
				err,
			)
	}

	if info.Size() <= 0 {
		return pdfOptimizationResult{},
			fmt.Errorf(
				"optimized PDF is empty",
			)
	}

	/*
		Bei normaler Optimierung verwenden wir das Original,
		wenn qpdf die Datei nicht verkleinern konnte.

		Bei Linearization gilt das bewusst nicht:
		Fast Web View kann die Datei geringfügig vergrößern,
		erfüllt aber trotzdem den gewünschten Zweck.
	*/
	if !linearize &&
		info.Size() >=
			upload.Size {
		if err :=
			writePDFOptimizationUnchangedMarker(
				upload,
			); err != nil {
			return pdfOptimizationResult{},
				err
		}

		return pdfOptimizationResult{
			Path: upload.Path,

			OriginalSize: upload.Size,

			ResultSize: upload.Size,

			Unchanged: true,
		}, nil
	}

	cachePath :=
		pdfOptimizationCachePath(
			upload,
			linearize,
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
			return pdfOptimizationResult{
				Path: cachePath,

				OriginalSize: upload.Size,

				ResultSize: existing.Size(),

				Linearized: linearize,
			}, nil
		}

		return pdfOptimizationResult{},
			fmt.Errorf(
				"cache optimized PDF: %w",
				err,
			)
	}

	return pdfOptimizationResult{
		Path: cachePath,

		OriginalSize: upload.Size,

		ResultSize: info.Size(),

		Linearized: linearize,
	}, nil
}

func (s *Server) optimizePDF(
	ctx context.Context,
	input string,
	output string,
	linearize bool,
) error {
	if linearize {
		return s.qpdf.
			OptimizeLinearized(
				ctx,
				input,
				output,
			)
	}

	return s.qpdf.Optimize(
		ctx,
		input,
		output,
	)
}

func readCachedPDFOptimizationResult(
	upload storedPDFUpload,
	linearize bool,
) (pdfOptimizationResult, bool) {
	cachePath :=
		pdfOptimizationCachePath(
			upload,
			linearize,
		)

	info, err :=
		os.Stat(
			cachePath,
		)

	if err == nil &&
		info.Size() > 0 {
		return pdfOptimizationResult{
			Path: cachePath,

			OriginalSize: upload.Size,

			ResultSize: info.Size(),

			Linearized: linearize,
		}, true
	}

	if !linearize {
		if _, err :=
			os.Stat(
				pdfOptimizationUnchangedMarkerPath(
					upload,
				),
			); err == nil {
			return pdfOptimizationResult{
				Path: upload.Path,

				OriginalSize: upload.Size,

				ResultSize: upload.Size,

				Unchanged: true,
			}, true
		}
	}

	return pdfOptimizationResult{},
		false
}

func pdfOptimizationCacheDirectory(
	upload storedPDFUpload,
) string {
	return filepath.Join(
		upload.Directory,
		"optimization",
	)
}

func pdfOptimizationCachePath(
	upload storedPDFUpload,
	linearize bool,
) string {
	name :=
		"optimized.pdf"

	if linearize {
		name =
			"linearized.pdf"
	}

	return filepath.Join(
		pdfOptimizationCacheDirectory(
			upload,
		),
		name,
	)
}

func pdfOptimizationUnchangedMarkerPath(
	upload storedPDFUpload,
) string {
	return filepath.Join(
		pdfOptimizationCacheDirectory(
			upload,
		),
		"optimized.unchanged",
	)
}

func writePDFOptimizationUnchangedMarker(
	upload storedPDFUpload,
) error {
	directory :=
		pdfOptimizationCacheDirectory(
			upload,
		)

	if err :=
		os.MkdirAll(
			directory,
			0700,
		); err != nil {
		return fmt.Errorf(
			"create optimization marker directory: %w",
			err,
		)
	}

	if err :=
		os.WriteFile(
			pdfOptimizationUnchangedMarkerPath(
				upload,
			),
			[]byte(
				"unchanged",
			),
			0600,
		); err != nil {
		return fmt.Errorf(
			"write optimization marker: %w",
			err,
		)
	}

	return nil
}
