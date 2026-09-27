package web

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"

	"github.com/jayzone91/media-converter/internal/media"
)

func (s *Server) validateConversionInput(
	ctx context.Context,
	format media.Format,
	path string,
) error {
	switch format.Category {
	case media.CategoryImage:
		if format.ID == "svg" {
			return validateSVGFile(
				path,
			)
		}

		if err := s.imageMagick.Validate(
			ctx,
			path,
		); err != nil {
			return fmt.Errorf(
				"invalid %s image: %w",
				format.ID,
				err,
			)
		}

	case media.CategoryPDF:
		if _, err := s.qpdf.PageCount(
			ctx,
			path,
		); err != nil {
			return fmt.Errorf(
				"invalid PDF: %w",
				err,
			)
		}

	/*
		Audio und Video wurden bereits vollständig
		durch ffprobe erkannt.

		Office-Dokumente wurden zuvor über ihre
		Container-Struktur erkannt.

		TXT und Markdown wurden über MIME und Inhalt
		validiert.
	*/
	case media.CategoryAudio,
		media.CategoryVideo,
		media.CategoryDocument,
		media.CategoryMarkdown:
		return nil
	}

	return nil
}

func validateSVGFile(
	path string,
) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := xml.NewDecoder(
		file,
	)

	foundRoot := false

	for {
		token, err := decoder.Token()

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf(
				"invalid SVG XML: %w",
				err,
			)
		}

		start, ok := token.(xml.StartElement)

		if !ok {
			continue
		}

		if !foundRoot {
			foundRoot = true

			if start.Name.Local != "svg" {
				return fmt.Errorf(
					"SVG root element is %q",
					start.Name.Local,
				)
			}
		}
	}

	if !foundRoot {
		return fmt.Errorf(
			"SVG contains no root element",
		)
	}

	return nil
}
