package web

import (
	"fmt"
	"image"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	_ "image/jpeg"
	_ "image/png"

	"github.com/jayzone91/media-converter/internal/converter"
)

const (
	maxPDFEditImageSize int64 = 32 << 20

	maxPDFEditImages = 20

	maxPDFEditImagePixels = 50_000_000
)

func savePDFEditImages(
	tempDir string,
	headers []*multipart.FileHeader,
	requests []pdfImageEditRequest,
	pageCount int,
) ([]converter.PDFImageEdit, error) {
	edits := make(
		[]converter.PDFImageEdit,
		0,
		len(headers),
	)

	for index, header := range headers {
		request :=
			requests[index]

		if err := validatePDFImageEditRequest(
			request,
			pageCount,
		); err != nil {
			return nil, err
		}

		path, err :=
			savePDFEditImage(
				tempDir,
				header,
				index,
			)
		if err != nil {
			return nil, err
		}

		edits = append(
			edits,
			converter.PDFImageEdit{
				Page: request.Page,

				Path: path,

				X: request.X,
				Y: request.Y,

				Width: request.Width,
			},
		)
	}

	return edits, nil
}

func savePDFEditImage(
	tempDir string,
	header *multipart.FileHeader,
	index int,
) (string, error) {
	if header.Size <= 0 {
		return "",
			fmt.Errorf(
				"image is empty",
			)
	}

	if header.Size >
		maxPDFEditImageSize {
		return "",
			fmt.Errorf(
				"image exceeds maximum size",
			)
	}

	file, err := header.Open()
	if err != nil {
		return "",
			fmt.Errorf(
				"open image upload: %w",
				err,
			)
	}
	defer file.Close()

	tempPath := filepath.Join(
		tempDir,
		fmt.Sprintf(
			"image-%03d.upload",
			index+1,
		),
	)

	if err := writePDFEditImage(
		tempPath,
		file,
	); err != nil {
		return "", err
	}

	config, format, err :=
		readPDFEditImage(
			tempPath,
		)
	if err != nil {
		return "", err
	}

	pixels :=
		int64(config.Width) *
			int64(config.Height)

	if pixels >
		maxPDFEditImagePixels {
		return "",
			fmt.Errorf(
				"image dimensions are too large",
			)
	}

	extension, err :=
		pdfEditImageExtension(
			format,
		)
	if err != nil {
		return "", err
	}

	finalPath :=
		strings.TrimSuffix(
			tempPath,
			filepath.Ext(
				tempPath,
			),
		) +
			extension

	if err := os.Rename(
		tempPath,
		finalPath,
	); err != nil {
		return "",
			fmt.Errorf(
				"prepare image upload: %w",
				err,
			)
	}

	return finalPath, nil
}

func writePDFEditImage(
	path string,
	source io.Reader,
) error {
	output, err := os.OpenFile(
		path,
		os.O_CREATE|
			os.O_WRONLY|
			os.O_TRUNC,
		0600,
	)
	if err != nil {
		return fmt.Errorf(
			"create image file: %w",
			err,
		)
	}

	written, copyErr :=
		io.Copy(
			output,
			io.LimitReader(
				source,
				maxPDFEditImageSize+1,
			),
		)

	closeErr :=
		output.Close()

	if copyErr != nil {
		return fmt.Errorf(
			"save image upload: %w",
			copyErr,
		)
	}

	if closeErr != nil {
		return fmt.Errorf(
			"close image upload: %w",
			closeErr,
		)
	}

	if written >
		maxPDFEditImageSize {
		return fmt.Errorf(
			"image exceeds maximum size",
		)
	}

	return nil
}

func readPDFEditImage(
	path string,
) (image.Config, string, error) {
	file, err := os.Open(
		path,
	)
	if err != nil {
		return image.Config{},
			"",
			fmt.Errorf(
				"open image: %w",
				err,
			)
	}
	defer file.Close()

	config, format, err :=
		image.DecodeConfig(
			file,
		)
	if err != nil {
		return image.Config{},
			"",
			fmt.Errorf(
				"invalid image: %w",
				err,
			)
	}

	if config.Width <= 0 ||
		config.Height <= 0 {
		return image.Config{},
			"",
			fmt.Errorf(
				"invalid image dimensions",
			)
	}

	return config,
		format,
		nil
}

func pdfEditImageExtension(
	format string,
) (string, error) {
	switch format {
	case "jpeg":
		return ".jpg", nil

	case "png":
		return ".png", nil

	default:
		return "",
			fmt.Errorf(
				"unsupported image format",
			)
	}
}
