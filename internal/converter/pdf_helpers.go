package converter

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func pdfPageNumber(
	path string,
) int {
	name := strings.TrimSuffix(
		filepath.Base(path),
		filepath.Ext(path),
	)

	index := strings.LastIndex(
		name,
		"-",
	)
	if index == -1 {
		return 0
	}

	number, err := strconv.Atoi(
		name[index+1:],
	)
	if err != nil {
		return 0
	}

	return number
}

func findTessdataDirectory(
	tesseract string,
) (string, error) {
	candidates := []string{
		os.Getenv(
			"TESSDATA_PREFIX",
		),

		filepath.Join(
			filepath.Dir(tesseract),
			"tessdata",
		),

		"/usr/share/tesseract-ocr/5/tessdata",
		"/usr/share/tesseract-ocr/4.00/tessdata",
		"/usr/share/tessdata",
	}

	for _, candidate := range candidates {
		candidate = strings.TrimSpace(
			candidate,
		)

		if candidate == "" {
			continue
		}

		deu := filepath.Join(
			candidate,
			"deu.traineddata",
		)

		eng := filepath.Join(
			candidate,
			"eng.traineddata",
		)

		if _, err := os.Stat(deu); err != nil {
			continue
		}

		if _, err := os.Stat(eng); err != nil {
			continue
		}

		return candidate, nil
	}

	return "", fmt.Errorf(
		"tesseract language data not found: deu.traineddata and eng.traineddata are required",
	)
}
