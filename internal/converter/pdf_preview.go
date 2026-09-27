package converter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

func (c *PDF) RenderPreviews(
	ctx context.Context,
	input string,
	outputDir string,
) ([]string, error) {
	if err := os.MkdirAll(
		outputDir,
		0700,
	); err != nil {
		return nil, fmt.Errorf(
			"failed to create PDF preview directory: %w",
			err,
		)
	}

	prefix := filepath.Join(
		outputDir,
		"page",
	)

	cmd := exec.CommandContext(
		ctx,
		c.pdfToPPM,
		"-jpeg",
		"-jpegopt",
		"quality=75",
		"-r",
		"90",
		input,
		prefix,
	)

	result, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(
			"pdftoppm preview rendering failed: %w: %s",
			err,
			string(result),
		)
	}

	pages, err := filepath.Glob(
		prefix + "-*.jpg",
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to find rendered PDF previews: %w",
			err,
		)
	}

	if len(pages) == 0 {
		return nil, fmt.Errorf(
			"pdftoppm produced no preview pages",
		)
	}

	sort.Slice(
		pages,
		func(i int, j int) bool {
			return pdfPageNumber(
				pages[i],
			) <
				pdfPageNumber(
					pages[j],
				)
		},
	)

	return pages, nil
}
