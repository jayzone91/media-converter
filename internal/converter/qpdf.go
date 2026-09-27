package converter

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type QPDF struct {
	binary string
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

	cmd := exec.CommandContext(
		ctx,
		q.binary,
		args...,
	)

	result, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"qpdf merge failed: %w: %s",
			err,
			string(result),
		)
	}

	return nil
}

func (q *QPDF) PageCount(
	ctx context.Context,
	input string,
) (int, error) {
	cmd := exec.CommandContext(
		ctx,
		q.binary,
		"--show-npages",
		input,
	)

	result, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf(
			"qpdf page count failed: %w: %s",
			err,
			strings.TrimSpace(
				string(result),
			),
		)
	}

	count, err := strconv.Atoi(
		strings.TrimSpace(
			string(result),
		),
	)
	if err != nil {
		return 0, fmt.Errorf(
			"invalid qpdf page count: %w",
			err,
		)
	}

	if count < 1 {
		return 0, fmt.Errorf(
			"PDF contains no pages",
		)
	}

	return count, nil
}
