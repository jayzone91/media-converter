package web

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type convertedArchiveFile struct {
	Path string
	Name string
}

func createConversionArchive(
	outputPath string,
	files []convertedArchiveFile,
) error {
	output, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf(
			"create conversion archive: %w",
			err,
		)
	}

	writer := zip.NewWriter(output)

	closeWithError := func() {
		_ = writer.Close()
		_ = output.Close()
	}

	usedNames := make(
		map[string]int,
		len(files),
	)

	for _, file := range files {
		name := uniqueArchiveFilename(
			file.Name,
			usedNames,
		)

		if err := addConversionArchiveFile(
			writer,
			file.Path,
			name,
		); err != nil {
			closeWithError()

			return err
		}
	}

	if err := writer.Close(); err != nil {
		_ = output.Close()

		return fmt.Errorf(
			"close conversion archive: %w",
			err,
		)
	}

	if err := output.Close(); err != nil {
		return fmt.Errorf(
			"close conversion archive file: %w",
			err,
		)
	}

	return nil
}

func addConversionArchiveFile(
	writer *zip.Writer,
	path string,
	name string,
) error {
	source, err := os.Open(path)
	if err != nil {
		return fmt.Errorf(
			"open converted file: %w",
			err,
		)
	}
	defer source.Close()

	destination, err := writer.Create(
		name,
	)
	if err != nil {
		return fmt.Errorf(
			"create archive entry: %w",
			err,
		)
	}

	if _, err := io.Copy(
		destination,
		source,
	); err != nil {
		return fmt.Errorf(
			"write archive entry: %w",
			err,
		)
	}

	return nil
}

func uniqueArchiveFilename(
	name string,
	used map[string]int,
) string {
	count := used[name]
	used[name] = count + 1

	if count == 0 {
		return name
	}

	extension := filepath.Ext(name)

	baseName := name[:len(name)-len(extension)]

	return fmt.Sprintf(
		"%s-%d%s",
		baseName,
		count+1,
		extension,
	)
}
