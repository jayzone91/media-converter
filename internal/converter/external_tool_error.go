package converter

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

const externalToolOutputLimit = 16 << 10

type ExternalToolError struct {
	Tool   string
	Err    error
	Output string
}

func (e *ExternalToolError) Error() string {
	if e.Output == "" {
		return fmt.Sprintf(
			"%s failed: %v",
			e.Tool,
			e.Err,
		)
	}

	return fmt.Sprintf(
		"%s failed: %v: %s",
		e.Tool,
		e.Err,
		e.Output,
	)
}

func (e *ExternalToolError) Unwrap() error {
	return e.Err
}

type limitedToolOutput struct {
	mu sync.Mutex

	data      []byte
	limit     int
	truncated bool
}

func newLimitedToolOutput(
	limit int,
) *limitedToolOutput {
	return &limitedToolOutput{
		data: make(
			[]byte,
			0,
			limit,
		),
		limit: limit,
	}
}

func (w *limitedToolOutput) Write(
	data []byte,
) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	originalLength := len(data)

	if w.limit <= 0 {
		if originalLength > 0 {
			w.truncated = true
		}

		return originalLength, nil
	}

	remaining :=
		w.limit -
			len(w.data)

	if remaining <= 0 {
		if originalLength > 0 {
			w.truncated = true
		}

		return originalLength, nil
	}

	if len(data) > remaining {
		w.data = append(
			w.data,
			data[:remaining]...,
		)

		w.truncated = true

		return originalLength, nil
	}

	w.data = append(
		w.data,
		data...,
	)

	return originalLength, nil
}

func (w *limitedToolOutput) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()

	output :=
		strings.TrimSpace(
			string(w.data),
		)

	if !w.truncated {
		return output
	}

	if output == "" {
		return "[Ausgabe gekürzt]"
	}

	return output +
		"\n[Ausgabe gekürzt]"
}

func limitExternalToolOutput(
	output string,
) string {
	output =
		strings.TrimSpace(
			output,
		)

	if len(output) <=
		externalToolOutputLimit {
		return output
	}

	output =
		strings.ToValidUTF8(
			output[:externalToolOutputLimit],
			"�",
		)

	return strings.TrimSpace(
		output,
	) + "\n[Ausgabe gekürzt]"
}

func newExternalToolError(
	ctx context.Context,
	tool string,
	err error,
	output string,
) error {
	if err == nil {
		return nil
	}

	if ctxErr :=
		ctx.Err(); ctxErr != nil {
		err = ctxErr
	}

	return &ExternalToolError{
		Tool: tool,

		Err: err,

		Output: output,
	}
}

func runExternalTool(
	ctx context.Context,
	tool string,
	cmd *exec.Cmd,
) error {
	output :=
		newLimitedToolOutput(
			externalToolOutputLimit,
		)

	cmd.Stdout = output
	cmd.Stderr = output

	err := cmd.Run()

	if err == nil {
		return nil
	}

	return newExternalToolError(
		ctx,
		tool,
		err,
		output.String(),
	)
}

func runExternalToolCapture(
	ctx context.Context,
	tool string,
	cmd *exec.Cmd,
) (
	string,
	string,
	error,
) {
	var stdout bytes.Buffer

	stderr :=
		newLimitedToolOutput(
			externalToolOutputLimit,
		)

	cmd.Stdout = &stdout
	cmd.Stderr = stderr

	err := cmd.Run()

	stdoutValue :=
		stdout.String()

	stderrValue :=
		stderr.String()

	if err == nil {
		return stdoutValue,
			stderrValue,
			nil
	}

	diagnostic :=
		stderrValue

	if diagnostic == "" {
		diagnostic =
			limitExternalToolOutput(
				stdoutValue,
			)
	}

	return stdoutValue,
		stderrValue,
		newExternalToolError(
			ctx,
			tool,
			err,
			diagnostic,
		)
}
