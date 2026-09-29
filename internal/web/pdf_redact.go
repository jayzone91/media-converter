package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jayzone91/media-converter/internal/converter"
)

const maxPDFRedactRequestSize int64 = 1 << 20

func (s *Server) handlePDFRedact(
	w http.ResponseWriter,
	r *http.Request,
) {
	r.Body =
		http.MaxBytesReader(
			w,
			r.Body,
			maxPDFRedactRequestSize,
		)

	request, err :=
		decodePDFRedactRequestBody(
			r,
		)

	if err != nil {
		s.logWarn(
			r,
			"PDF redaction rejected",
			"reason",
			err.Error(),
		)

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Ungültige Schwärzungsdaten.",
		)

		return
	}

	upload, ok :=
		s.pdfUploads.Get(
			request.UploadID,
		)

	if !ok {
		writeAPIError(
			w,
			http.StatusGone,
			apiErrorUploadExpired,
			"Die PDF ist nicht mehr verfügbar. Bitte erneut hochladen.",
		)

		return
	}

	redactions, err :=
		validatePDFRedactRequest(
			request,
			upload.PageCount,
		)

	if err != nil {
		s.logWarn(
			r,
			"PDF redaction rejected",
			"reason",
			err.Error(),
			"upload_id",
			upload.ID,
		)

		writeAPIError(
			w,
			http.StatusBadRequest,
			apiErrorInvalidRequest,
			"Die Schwärzungsbereiche sind ungültig.",
		)

		return
	}

	releaseWorkloads, err :=
		s.acquireConversionWorkloads(
			r.Context(),
			conversionWorkloadPlan{
				Workloads: []workloadType{
					workloadPDFCPU,
					workloadPoppler,
					workloadQPDF,
				},
			},
		)

	if err != nil {
		s.logError(
			r,
			"PDF redaction queue failed",
			err,
			"upload_id",
			upload.ID,
		)

		writeQueueAPIError(
			w,
			err,
		)

		return
	}

	defer releaseWorkloads()

	tempDir, err :=
		os.MkdirTemp(
			"",
			"media-converter-pdf-redact-*",
		)

	if err != nil {
		s.logError(
			r,
			"PDF redaction temporary directory failed",
			err,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Temporäres Verzeichnis konnte nicht erstellt werden.",
		)

		return
	}

	defer func() {
		if err := os.RemoveAll(
			tempDir,
		); err != nil {
			s.logError(
				r,
				"PDF redaction cleanup failed",
				err,
				"directory",
				tempDir,
			)
		}
	}()

	replacements :=
		make(
			map[int]string,
			len(redactions),
		)

	for page, pageRedactions := range redactions {
		pagePath :=
			filepath.Join(
				tempDir,
				fmt.Sprintf(
					"page-%06d.pdf",
					page,
				),
			)

		if err :=
			s.pdf.RenderRedactedPage(
				r.Context(),
				upload.Path,
				pagePath,
				page,
				pageRedactions,
			); err != nil {
			if requestContextEnded(
				r.Context(),
			) {
				return
			}

			s.logError(
				r,
				"PDF redaction page rendering failed",
				err,
				"upload_id",
				upload.ID,
				"page",
				page,
			)

			writeAPIError(
				w,
				http.StatusInternalServerError,
				apiErrorInternal,
				"Die PDF konnte nicht sicher geschwärzt werden.",
			)

			return
		}

		replacements[page] =
			pagePath
	}

	assembledPath :=
		filepath.Join(
			tempDir,
			"assembled.pdf",
		)

	if err :=
		s.qpdf.AssembleRedactedPDF(
			r.Context(),
			upload.Path,
			replacements,
			upload.PageCount,
			assembledPath,
		); err != nil {
		if requestContextEnded(
			r.Context(),
		) {
			return
		}

		s.logError(
			r,
			"PDF redaction assembly failed",
			err,
			"upload_id",
			upload.ID,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Die geschwärzte PDF konnte nicht erstellt werden.",
		)

		return
	}

	attachmentFreePath :=
		filepath.Join(
			tempDir,
			"without-attachments.pdf",
		)

	removedAttachments, err :=
		converter.RemoveAllPDFAttachments(
			assembledPath,
			attachmentFreePath,
		)

	if err != nil {
		s.logError(
			r,
			"PDF redaction attachment cleanup failed",
			err,
			"upload_id",
			upload.ID,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Versteckte PDF-Inhalte konnten nicht sicher entfernt werden.",
		)

		return
	}

	outputPath :=
		filepath.Join(
			tempDir,
			"geschwaerzt.pdf",
		)

	if err :=
		s.qpdf.SanitizeRedactedPDF(
			r.Context(),
			attachmentFreePath,
			outputPath,
		); err != nil {
		if requestContextEnded(
			r.Context(),
		) {
			return
		}

		s.logError(
			r,
			"PDF redaction sanitization failed",
			err,
			"upload_id",
			upload.ID,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Die geschwärzte PDF konnte nicht sicher bereinigt werden.",
		)

		return
	}

	if err :=
		s.qpdf.CheckPDF(
			r.Context(),
			outputPath,
		); err != nil {
		if requestContextEnded(
			r.Context(),
		) {
			return
		}

		s.logError(
			r,
			"PDF redaction output validation failed",
			err,
			"upload_id",
			upload.ID,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Die erzeugte PDF hat die Sicherheitsprüfung nicht bestanden.",
		)

		return
	}

	outputSize, err :=
		downloadFileSize(
			outputPath,
		)

	if err != nil {
		s.logError(
			r,
			"PDF redaction output inspection failed",
			err,
			"upload_id",
			upload.ID,
		)

		writeAPIError(
			w,
			http.StatusInternalServerError,
			apiErrorInternal,
			"Die erzeugte PDF konnte nicht gelesen werden.",
		)

		return
	}

	_, ok =
		s.preparePDFRedactResult(
			w,
			r,
			outputPath,
			outputSize,
			upload.PageCount,
		)

	if !ok {
		return
	}

	s.pdfUploads.Delete(
		upload.ID,
	)

	s.logInfo(
		"PDF redacted",
		"pages",
		len(redactions),
		"areas",
		len(request.Redactions),
		"attachments",
		removedAttachments,
		"input",
		upload.Size,
		"output",
		outputSize,
	)
}

func requestContextEnded(
	ctx context.Context,
) bool {
	return ctx.Err() != nil
}

func decodePDFRedactRequestBody(
	r *http.Request,
) (pdfRedactRequest, error) {
	defer r.Body.Close()

	data, err :=
		io.ReadAll(
			io.LimitReader(
				r.Body,
				maxPDFRedactRequestSize+1,
			),
		)

	if err != nil {
		return pdfRedactRequest{},
			fmt.Errorf(
				"read request body: %w",
				err,
			)
	}

	if int64(len(data)) >
		maxPDFRedactRequestSize {
		return pdfRedactRequest{},
			fmt.Errorf(
				"request too large",
			)
	}

	return decodePDFRedactRequest(
		string(data),
	)
}
