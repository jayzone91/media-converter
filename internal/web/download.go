package web

import (
	"mime"
	"net/http"
	"os"
)

func (s *Server) handleDownload(
	w http.ResponseWriter,
	r *http.Request,
) {
	id := r.PathValue(
		"id",
	)

	download, ok :=
		s.downloads.Take(
			id,
		)

	if !ok {
		http.Error(
			w,
			"Der Download ist nicht mehr verfügbar.",
			http.StatusGone,
		)

		return
	}

	defer func() {
		if err := os.RemoveAll(
			download.Directory,
		); err != nil {
			s.logError(
				r,
				"download cleanup failed",
				err,
				"filename",
				download.Filename,
			)
		}
	}()

	file, err := os.Open(
		download.Path,
	)
	if err != nil {
		s.logError(
			r,
			"download open failed",
			err,
			"filename",
			download.Filename,
		)

		http.Error(
			w,
			"Die Datei konnte nicht geöffnet werden.",
			http.StatusInternalServerError,
		)

		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		s.logError(
			r,
			"download stat failed",
			err,
			"filename",
			download.Filename,
		)

		http.Error(
			w,
			"Die Datei konnte nicht gelesen werden.",
			http.StatusInternalServerError,
		)

		return
	}

	disposition :=
		mime.FormatMediaType(
			"attachment",
			map[string]string{
				"filename": download.Filename,
			},
		)

	w.Header().Set(
		"Content-Disposition",
		disposition,
	)

	w.Header().Set(
		"Cache-Control",
		"no-store",
	)

	s.logInfo(
		"download",
		"filename",
		download.Filename,
		"size",
		info.Size(),
	)

	http.ServeContent(
		w,
		r,
		download.Filename,
		info.ModTime(),
		file,
	)
}
