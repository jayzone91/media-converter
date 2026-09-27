package web

import (
	"encoding/json"
	"net/http"
	"os"
)

type downloadResponse struct {
	DownloadURL string `json:"download_url"`

	Filename string `json:"filename"`
}

func (s *Server) prepareDownloadResponse(
	w http.ResponseWriter,
	r *http.Request,
	path string,
	filename string,
) bool {
	download, err :=
		s.downloads.Add(
			path,
			filename,
		)

	if err != nil {
		s.logError(
			r,
			"download preparation failed",
			err,
			"filename",
			filename,
		)

		http.Error(
			w,
			"Download konnte nicht vorbereitet werden.",
			http.StatusInternalServerError,
		)

		return false
	}

	response :=
		downloadResponse{
			DownloadURL: "/downloads/" +
				download.ID,

			Filename: download.Filename,
		}

	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	w.Header().Set(
		"Cache-Control",
		"no-store",
	)

	if err := json.NewEncoder(
		w,
	).Encode(
		response,
	); err != nil {
		s.logError(
			r,
			"download response failed",
			err,
			"filename",
			filename,
		)

		return false
	}

	return true
}

func downloadFileSize(
	path string,
) (int64, error) {
	info, err := os.Stat(
		path,
	)

	if err != nil {
		return 0, err
	}

	return info.Size(), nil
}
