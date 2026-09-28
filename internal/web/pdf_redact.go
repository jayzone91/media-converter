package web

import (
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

		http.Error(
			w,
			"Ungültige Schwärzungsdaten.",
			http.StatusBadRequest,
		)

		return
	}

	upload, ok :=
		s.pdfUploads.Get(
			request.UploadID,
		)

	if !ok {
		http.Error(
			w,
			"Die PDF ist nicht mehr verfügbar. Bitte erneut hochladen.",
			http.StatusGone,
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

		http.Error(
			w,
			"Die Schwärzungsbereiche sind ungültig.",
			http.StatusBadRequest,
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
		http.Error(
			w,
			"Der Server ist momentan ausgelastet. Bitte später erneut versuchen.",
			http.StatusServiceUnavailable,
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

		http.Error(
			w,
			"Temporäres Verzeichnis konnte nicht erstellt werden.",
			http.StatusInternalServerError,
		)

		return
	}

	defer os.RemoveAll(
		tempDir,
	)

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
			s.logError(
				r,
				"PDF redaction page rendering failed",
				err,
				"upload_id",
				upload.ID,
				"page",
				page,
			)

			http.Error(
				w,
				"Die PDF konnte nicht sicher geschwärzt werden.",
				http.StatusInternalServerError,
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
		s.logError(
			r,
			"PDF redaction assembly failed",
			err,
			"upload_id",
			upload.ID,
		)

		http.Error(
			w,
			"Die geschwärzte PDF konnte nicht erstellt werden.",
			http.StatusInternalServerError,
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

		http.Error(
			w,
			"Versteckte PDF-Inhalte konnten nicht sicher entfernt werden.",
			http.StatusInternalServerError,
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
		s.logError(
			r,
			"PDF redaction sanitization failed",
			err,
			"upload_id",
			upload.ID,
		)

		http.Error(
			w,
			"Die geschwärzte PDF konnte nicht sicher bereinigt werden.",
			http.StatusInternalServerError,
		)

		return
	}

	if err :=
		s.qpdf.CheckPDF(
			r.Context(),
			outputPath,
		); err != nil {
		s.logError(
			r,
			"PDF redaction output validation failed",
			err,
			"upload_id",
			upload.ID,
		)

		http.Error(
			w,
			"Die erzeugte PDF hat die Sicherheitsprüfung nicht bestanden.",
			http.StatusInternalServerError,
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

		http.Error(
			w,
			"Die erzeugte PDF konnte nicht gelesen werden.",
			http.StatusInternalServerError,
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
