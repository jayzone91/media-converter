package converter

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

type QPDF struct {
	binary string
}

type QPDFPageCountResult struct {
	Count    int
	Warnings string
}

type PDFPageRotation struct {
	Page  int
	Angle int
}

func NewQPDF() (*QPDF, error) {
	binary, err := exec.LookPath("qpdf")
	if err != nil {
		return nil, fmt.Errorf(
			"qpdf not found: %w",
			err,
		)
	}

	return &QPDF{
		binary: binary,
	}, nil
}

func (q *QPDF) Merge(
	ctx context.Context,
	inputs []string,
	output string,
) error {
	if len(inputs) < 2 {
		return fmt.Errorf(
			"at least two PDFs are required",
		)
	}

	args := []string{
		"--warning-exit-0",
		"--stream-data=preserve",
		"--empty",
		"--pages",
	}

	for _, input := range inputs {
		args = append(
			args,
			input,
			"1-z",
		)
	}

	args = append(
		args,
		"--",
		output,
	)

	return q.run(
		ctx,
		"merge",
		args,
	)
}

func (q *QPDF) Reorder(
	ctx context.Context,
	input string,
	pages []int,
	output string,
) error {
	if len(pages) == 0 {
		return fmt.Errorf(
			"at least one PDF page is required",
		)
	}

	pageParts := make(
		[]string,
		len(pages),
	)

	for index, page := range pages {
		if page < 1 {
			return fmt.Errorf(
				"invalid PDF page number: %d",
				page,
			)
		}

		pageParts[index] = strconv.Itoa(
			page,
		)
	}

	args := []string{
		"--warning-exit-0",
		"--stream-data=preserve",
		"--empty",
		"--pages",
		input,
		strings.Join(
			pageParts,
			",",
		),
		"--",
		output,
	}

	return q.run(
		ctx,
		"reorder",
		args,
	)
}

func (q *QPDF) RotatePages(
	ctx context.Context,
	input string,
	rotations []PDFPageRotation,
	output string,
) error {
	grouped, err := groupPDFRotations(
		rotations,
	)
	if err != nil {
		return err
	}

	if len(grouped) == 0 {
		return fmt.Errorf(
			"at least one page rotation is required",
		)
	}

	args := []string{
		"--warning-exit-0",
		"--stream-data=preserve",
		input,
	}

	for _, angle := range []int{
		90,
		180,
		270,
	} {
		pages := grouped[angle]
		if len(pages) == 0 {
			continue
		}

		pageRange, err := qpdfPageRange(
			pages,
		)
		if err != nil {
			return err
		}

		rotation, err := qpdfRotation(
			angle,
		)
		if err != nil {
			return err
		}

		args = append(
			args,
			fmt.Sprintf(
				"--rotate=%s:%s",
				rotation,
				pageRange,
			),
		)
	}

	args = append(
		args,
		output,
	)

	return q.run(
		ctx,
		"rotate",
		args,
	)
}

func (q *QPDF) PageCount(
	ctx context.Context,
	input string,
) (QPDFPageCountResult, error) {
	cmd := externalCommandContext(
		ctx,
		q.binary,
		"--warning-exit-0",
		"--show-npages",
		input,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return QPDFPageCountResult{},
				fmt.Errorf(
					"qpdf page count failed: %w",
					ctxErr,
				)
		}

		return QPDFPageCountResult{},
			fmt.Errorf(
				"qpdf page count failed: %w: %s",
				err,
				strings.TrimSpace(
					stderr.String(),
				),
			)
	}

	output := strings.TrimSpace(
		stdout.String(),
	)

	count, err := strconv.Atoi(
		output,
	)
	if err != nil {
		return QPDFPageCountResult{},
			fmt.Errorf(
				"invalid qpdf page count %q: %w",
				output,
				err,
			)
	}

	if count < 1 {
		return QPDFPageCountResult{},
			fmt.Errorf(
				"PDF contains no pages",
			)
	}

	return QPDFPageCountResult{
		Count: count,

		Warnings: strings.TrimSpace(
			stderr.String(),
		),
	}, nil
}

func (q *QPDF) run(
	ctx context.Context,
	operation string,
	args []string,
) error {
	cmd := externalCommandContext(
		ctx,
		q.binary,
		args...,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf(
				"qpdf %s failed: %w",
				operation,
				ctxErr,
			)
		}

		message := strings.TrimSpace(
			stderr.String(),
		)

		if message == "" {
			message = strings.TrimSpace(
				stdout.String(),
			)
		}

		return fmt.Errorf(
			"qpdf %s failed: %w: %s",
			operation,
			err,
			message,
		)
	}

	return nil
}

func groupPDFRotations(
	rotations []PDFPageRotation,
) (map[int][]int, error) {
	grouped := map[int][]int{
		90:  {},
		180: {},
		270: {},
	}

	seen := make(
		map[int]struct{},
		len(rotations),
	)

	for _, rotation := range rotations {
		if rotation.Page < 1 {
			return nil, fmt.Errorf(
				"invalid PDF page number: %d",
				rotation.Page,
			)
		}

		if _, exists := seen[rotation.Page]; exists {
			return nil, fmt.Errorf(
				"duplicate PDF page number: %d",
				rotation.Page,
			)
		}

		seen[rotation.Page] = struct{}{}

		if _, err := qpdfRotation(
			rotation.Angle,
		); err != nil {
			return nil, err
		}

		grouped[rotation.Angle] = append(
			grouped[rotation.Angle],
			rotation.Page,
		)
	}

	return grouped, nil
}

func qpdfRotation(
	angle int,
) (string, error) {
	switch angle {
	case 90:
		return "+90", nil

	case 180:
		return "+180", nil

	case 270:
		return "+270", nil

	default:
		return "", fmt.Errorf(
			"unsupported PDF rotation angle: %d",
			angle,
		)
	}
}

func qpdfPageRange(
	pages []int,
) (string, error) {
	if len(pages) == 0 {
		return "", fmt.Errorf(
			"at least one PDF page is required",
		)
	}

	sorted := append(
		[]int(nil),
		pages...,
	)

	sort.Ints(
		sorted,
	)

	for index, page := range sorted {
		if page < 1 {
			return "", fmt.Errorf(
				"invalid PDF page number: %d",
				page,
			)
		}

		if index > 0 &&
			page == sorted[index-1] {
			return "", fmt.Errorf(
				"duplicate PDF page number: %d",
				page,
			)
		}
	}

	var ranges []string

	start := sorted[0]
	end := start

	for _, page := range sorted[1:] {
		if page == end+1 {
			end = page

			continue
		}

		ranges = append(
			ranges,
			formatQPDFRange(
				start,
				end,
			),
		)

		start = page
		end = page
	}

	ranges = append(
		ranges,
		formatQPDFRange(
			start,
			end,
		),
	)

	return strings.Join(
		ranges,
		",",
	), nil
}

func formatQPDFRange(
	start int,
	end int,
) string {
	if start == end {
		return strconv.Itoa(
			start,
		)
	}

	return fmt.Sprintf(
		"%d-%d",
		start,
		end,
	)
}
