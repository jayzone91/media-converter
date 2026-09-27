package web

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

func validatePDFUpload(
	header *multipart.FileHeader,
) error {
	if header.Size <= 0 {
		return fmt.Errorf(
			"Die PDF-Datei ist leer.",
		)
	}

	if header.Size > maxFileSize {
		return fmt.Errorf(
			"Eine einzelne PDF darf maximal 512 MiB groß sein.",
		)
	}

	extension := strings.ToLower(
		filepath.Ext(
			header.Filename,
		),
	)

	if extension != ".pdf" {
		return fmt.Errorf(
			"Es können nur PDF-Dateien hochgeladen werden.",
		)
	}

	return nil
}

func savePDFUpload(
	source multipart.File,
	path string,
) (int64, error) {
	destination, err := os.OpenFile(
		path,
		os.O_WRONLY|
			os.O_CREATE|
			os.O_EXCL,
		0600,
	)
	if err != nil {
		return 0, err
	}

	limited := io.LimitReader(
		source,
		maxFileSize+1,
	)

	written, copyErr := io.Copy(
		destination,
		limited,
	)

	closeErr := destination.Close()

	if copyErr != nil {
		return 0, copyErr
	}

	if closeErr != nil {
		return 0, closeErr
	}

	if written > maxFileSize {
		return 0, fmt.Errorf(
			"PDF exceeds maximum size",
		)
	}

	return written, nil
}

func validatePDFSignature(
	path string,
) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	buffer := make(
		[]byte,
		1024,
	)

	n, err := file.Read(buffer)
	if err != nil &&
		!errors.Is(err, io.EOF) {
		return err
	}

	if !bytes.Contains(
		buffer[:n],
		[]byte("%PDF-"),
	) {
		return fmt.Errorf(
			"PDF signature not found",
		)
	}

	return nil
}
