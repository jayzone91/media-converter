package converter

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

func (c *PDF) ConvertToImages(
	ctx context.Context,
	input string,
	output string,
	format string,
) error {
	switch format {
	case "png", "jpeg":
	default:
		return fmt.Errorf(
			"unsupported PDF image format: %s",
			format,
		)
	}

	tempDir, err := os.MkdirTemp(
		filepath.Dir(input),
		"pdf-pages-*",
	)
	if err != nil {
		return fmt.Errorf(
			"failed to create PDF image temp dir: %w",
			err,
		)
	}
	defer os.RemoveAll(tempDir)

	prefix := filepath.Join(
		tempDir,
		"page",
	)

	args := []string{
		"-r",
		"200",
	}

	switch format {
	case "png":
		args = append(
			args,
			"-png",
		)

	case "jpeg":
		args = append(
			args,
			"-jpeg",
			"-jpegopt",
			"quality=90",
		)
	}

	args = append(
		args,
		input,
		prefix,
	)

	cmd := externalCommandContext(
		ctx,
		c.pdfToPPM,
		args...,
	)

	if err :=
		runExternalTool(
			ctx,
			"pdftoppm",
			cmd,
		); err != nil {
		return err
	}

	extension := "." + format
	if format == "jpeg" {
		extension = ".jpg"
	}

	pages, err := filepath.Glob(
		prefix + "-*" + extension,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to find rendered PDF pages: %w",
			err,
		)
	}

	if len(pages) == 0 {
		return fmt.Errorf(
			"pdftoppm produced no pages",
		)
	}

	sortPDFPagePaths(pages)

	if err := createImageZIP(
		output,
		pages,
		extension,
	); err != nil {
		return fmt.Errorf(
			"failed to create PDF image archive: %w",
			err,
		)
	}

	return nil
}

func sortPDFPagePaths(
	pages []string,
) {
	sort.Slice(
		pages,
		func(i, j int) bool {
			return pdfPageNumber(
				pages[i],
			) < pdfPageNumber(
				pages[j],
			)
		},
	)
}

func createImageZIP(
	output string,
	pages []string,
	extension string,
) error {
	file, err := os.Create(output)
	if err != nil {
		return err
	}

	archive := zip.NewWriter(file)

	for index, page := range pages {
		if err := addFileToZIP(
			archive,
			page,
			fmt.Sprintf(
				"page-%d%s",
				index+1,
				extension,
			),
		); err != nil {
			_ = archive.Close()
			_ = file.Close()

			return err
		}
	}

	if err := archive.Close(); err != nil {
		_ = file.Close()

		return err
	}

	if err := file.Close(); err != nil {
		return err
	}

	return nil
}

func addFileToZIP(
	archive *zip.Writer,
	path string,
	name string,
) error {
	source, err := os.Open(path)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := archive.Create(name)
	if err != nil {
		return err
	}

	if _, err := io.Copy(
		destination,
		source,
	); err != nil {
		return err
	}

	return nil
}
