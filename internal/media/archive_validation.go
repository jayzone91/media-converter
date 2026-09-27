package media

import (
	"archive/zip"
	"fmt"
	"io"
)

const (
	maxArchiveEntries          = 10_000
	maxArchiveUncompressedSize = uint64(1 << 30)
	maxArchiveEntrySize        = uint64(512 << 20)
	maxArchiveCompressionRatio = uint64(100)
)

func ValidateDocumentArchive(
	path string,
) error {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf(
			"open document archive: %w",
			err,
		)
	}
	defer reader.Close()

	if len(reader.File) == 0 {
		return fmt.Errorf(
			"document archive is empty",
		)
	}

	if len(reader.File) > maxArchiveEntries {
		return fmt.Errorf(
			"document archive contains too many entries: %d",
			len(reader.File),
		)
	}

	var totalUncompressed uint64

	for _, file := range reader.File {
		if err := validateArchiveEntry(
			file,
		); err != nil {
			return err
		}

		if file.UncompressedSize64 >
			maxArchiveUncompressedSize-totalUncompressed {
			return fmt.Errorf(
				"document archive exceeds maximum uncompressed size",
			)
		}

		totalUncompressed +=
			file.UncompressedSize64
	}

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		if err := validateArchiveEntryReadable(
			file,
		); err != nil {
			return err
		}
	}

	return nil
}

func validateArchiveEntry(
	file *zip.File,
) error {
	if file.UncompressedSize64 >
		maxArchiveEntrySize {
		return fmt.Errorf(
			"document archive entry %q exceeds maximum size",
			file.Name,
		)
	}

	if file.UncompressedSize64 == 0 {
		return nil
	}

	if file.CompressedSize64 == 0 {
		return fmt.Errorf(
			"document archive entry %q has invalid compressed size",
			file.Name,
		)
	}

	ratio :=
		file.UncompressedSize64 /
			file.CompressedSize64

	if ratio > maxArchiveCompressionRatio {
		return fmt.Errorf(
			"document archive entry %q exceeds maximum compression ratio",
			file.Name,
		)
	}

	return nil
}

func validateArchiveEntryReadable(
	file *zip.File,
) error {
	reader, err := file.Open()
	if err != nil {
		return fmt.Errorf(
			"open document archive entry %q: %w",
			file.Name,
			err,
		)
	}
	defer reader.Close()

	written, err := io.Copy(
		io.Discard,
		io.LimitReader(
			reader,
			int64(file.UncompressedSize64)+1,
		),
	)
	if err != nil {
		return fmt.Errorf(
			"read document archive entry %q: %w",
			file.Name,
			err,
		)
	}

	if uint64(written) !=
		file.UncompressedSize64 {
		return fmt.Errorf(
			"document archive entry %q has invalid size",
			file.Name,
		)
	}

	return nil
}
