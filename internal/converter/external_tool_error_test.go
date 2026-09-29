package converter

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestExternalToolErrorUnwrap(
	t *testing.T,
) {
	t.Parallel()

	baseErr :=
		errors.New(
			"test failure",
		)

	err :=
		&ExternalToolError{
			Tool: "test",

			Err: baseErr,

			Output: "details",
		}

	if !errors.Is(
		err,
		baseErr,
	) {
		t.Fatal(
			"ExternalToolError does not unwrap underlying error",
		)
	}
}

func TestLimitedToolOutputTruncates(
	t *testing.T,
) {
	t.Parallel()

	output :=
		newLimitedToolOutput(
			8,
		)

	written, err :=
		output.Write(
			[]byte(
				"abcdefghijkl",
			),
		)

	if err != nil {
		t.Fatalf(
			"Write returned error: %v",
			err,
		)
	}

	if written != 12 {
		t.Fatalf(
			"written = %d, want 12",
			written,
		)
	}

	result :=
		output.String()

	if !strings.HasPrefix(
		result,
		"abcdefgh",
	) {
		t.Fatalf(
			"unexpected output: %q",
			result,
		)
	}

	if !strings.Contains(
		result,
		"Ausgabe gekürzt",
	) {
		t.Fatalf(
			"truncation marker missing: %q",
			result,
		)
	}
}

func TestRunExternalToolPreservesCancellation(
	t *testing.T,
) {
	t.Parallel()

	ctx, cancel :=
		context.WithCancel(
			context.Background(),
		)

	cancel()

	cmd :=
		exec.CommandContext(
			ctx,
			"this-command-does-not-need-to-exist",
		)

	err :=
		runExternalTool(
			ctx,
			"test",
			cmd,
		)

	if !errors.Is(
		err,
		context.Canceled,
	) {
		t.Fatalf(
			"error = %v, want context.Canceled",
			err,
		)
	}
}
