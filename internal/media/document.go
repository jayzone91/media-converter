package media

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"strings"
)

func DetectDocument(path string) (Format, error) {
	file, err := os.Open(path)
	if err != nil {
		return Format{}, err
	}

	header := make([]byte, 5)

	n, readErr := file.Read(header)

	closeErr := file.Close()
	if closeErr != nil {
		return Format{}, closeErr
	}

	if readErr == nil || readErr == io.EOF {
		if string(header[:n]) == `{\rtf` {
			return Formats["rtf"], nil
		}
	}

	reader, err := zip.OpenReader(path)
	if err != nil {
		return Format{}, err
	}
	defer reader.Close()

	var (
		hasWord       bool
		hasExcel      bool
		hasPowerPoint bool
		odfMIME       string
	)

	for _, file := range reader.File {
		switch {
		case strings.HasPrefix(file.Name, "word/"):
			hasWord = true

		case strings.HasPrefix(file.Name, "xl/"):
			hasExcel = true

		case strings.HasPrefix(file.Name, "ppt/"):
			hasPowerPoint = true

		case file.Name == "mimetype":
			mime, err := readZipFile(file)
			if err != nil {
				return Format{}, err
			}

			odfMIME = strings.TrimSpace(string(mime))
		}
	}

	switch {
	case hasWord:
		return Formats["docx"], nil

	case hasExcel:
		return Formats["xlsx"], nil

	case hasPowerPoint:
		return Formats["pptx"], nil
	}

	switch odfMIME {
	case "application/vnd.oasis.opendocument.text":
		return Formats["odt"], nil

	case "application/vnd.oasis.opendocument.spreadsheet":
		return Formats["ods"], nil

	case "application/vnd.oasis.opendocument.presentation":
		return Formats["odp"], nil
	}

	return Format{}, fmt.Errorf("unsupported document format")
}

func readZipFile(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}
