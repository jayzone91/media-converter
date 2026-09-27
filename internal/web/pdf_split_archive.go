package web

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

type pdfPageRange struct {
	Start int
	End   int
}

func buildPDFSplitRanges(
	pageCount int,
	splitAfter []int,
) ([]pdfPageRange, error) {
	if pageCount < 2 {
		return nil, fmt.Errorf(
			"PDF requires at least two pages",
		)
	}

	if len(splitAfter) == 0 {
		return nil, fmt.Errorf(
			"at least one split point is required",
		)
	}

	points := append(
		[]int(nil),
		splitAfter...,
	)

	sort.Ints(points)

	for index, page := range points {
		if page < 1 ||
			page >= pageCount {
			return nil, fmt.Errorf(
				"split point %d is outside range 1-%d",
				page,
				pageCount-1,
			)
		}

		if index > 0 &&
			page == points[index-1] {
			return nil, fmt.Errorf(
				"duplicate split point: %d",
				page,
			)
		}
	}

	ranges := make(
		[]pdfPageRange,
		0,
		len(points)+1,
	)

	start := 1

	for _, end := range points {
		ranges = append(
			ranges,
			pdfPageRange{
				Start: start,
				End:   end,
			},
		)

		start = end + 1
	}

	ranges = append(
		ranges,
		pdfPageRange{
			Start: start,
			End:   pageCount,
		},
	)

	return ranges, nil
}

func createPDFSplitArchive(
	output string,
	files []string,
	ranges []pdfPageRange,
) error {
	if len(files) == 0 ||
		len(files) != len(ranges) {
		return fmt.Errorf(
			"invalid PDF split archive input",
		)
	}

	file, err := os.Create(
		output,
	)
	if err != nil {
		return err
	}

	archive := zip.NewWriter(
		file,
	)

	for index, path := range files {
		name := pdfSplitFilename(
			index+1,
			ranges[index],
		)

		if err := addPDFSplitFile(
			archive,
			path,
			name,
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

	return file.Close()
}

func pdfSplitFilename(
	index int,
	pageRange pdfPageRange,
) string {
	if pageRange.Start ==
		pageRange.End {
		return fmt.Sprintf(
			"teil-%03d-seite-%d.pdf",
			index,
			pageRange.Start,
		)
	}

	return fmt.Sprintf(
		"teil-%03d-seiten-%d-%d.pdf",
		index,
		pageRange.Start,
		pageRange.End,
	)
}

func addPDFSplitFile(
	archive *zip.Writer,
	path string,
	name string,
) error {
	source, err := os.Open(
		filepath.Clean(path),
	)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := archive.Create(
		name,
	)
	if err != nil {
		return err
	}

	_, err = io.Copy(
		destination,
		source,
	)

	return err
}
