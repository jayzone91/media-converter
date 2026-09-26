// Package media provides media type detection and format definitions.
package media

import (
	"fmt"
	"net/http"
	"os"
)

type Detection struct {
	MIME string
}

func Detect(path string) (Detection, error) {
	file, err := os.Open(path)
	if err != nil {
		return Detection{}, err
	}
	defer file.Close()

	buffer := make([]byte, 512)

	n, err := file.Read(buffer)
	if err != nil {
		return Detection{}, err
	}

	mime := http.DetectContentType(buffer[:n])

	if mime == "application/octet-stream" {
		return Detection{}, fmt.Errorf("unknown media type")
	}

	return Detection{
		MIME: mime,
	}, nil
}
