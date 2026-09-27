package converter

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
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

func NewQPDF() (*QPDF, error) {
	binary, err :=
		exec.LookPath(
			"qpdf",
		)
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

	cmd :=
		exec.CommandContext(
			ctx,
			q.binary,
			args...,
		)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctxErr :=
			ctx.Err(); ctxErr != nil {
			return fmt.Errorf(
				"qpdf merge failed: %w",
				ctxErr,
			)
		}

		return fmt.Errorf(
			"qpdf merge failed: %w: %s",
			err,
			strings.TrimSpace(
				stderr.String(),
			),
		)
	}

	return nil
}

func (q *QPDF) PageCount(
	ctx context.Context,
	input string,
) (QPDFPageCountResult, error) {
	cmd :=
		exec.CommandContext(
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
		if ctxErr :=
			ctx.Err(); ctxErr != nil {
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

	output :=
		strings.TrimSpace(
			stdout.String(),
		)

	count, err :=
		strconv.Atoi(
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
