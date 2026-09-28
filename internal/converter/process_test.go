package converter

import (
	"context"
	"errors"
	"os"
	"runtime"
	"testing"
	"time"
)

func TestExternalCommandContextFollowsCancellation(
	t *testing.T,
) {
	t.Parallel()

	ctx, cancel :=
		context.WithCancel(
			context.Background(),
		)

	var cmdName string
	var args []string

	if runtime.GOOS == "windows" {
		cmdName =
			"cmd"

		args =
			[]string{
				"/C",
				"ping -n 30 127.0.0.1 > nul",
			}
	} else {
		cmdName =
			"sh"

		args =
			[]string{
				"-c",
				"sleep 30",
			}
	}

	cmd :=
		externalCommandContext(
			ctx,
			cmdName,
			args...,
		)

	done :=
		make(
			chan error,
			1,
		)

	go func() {
		done <- cmd.Run()
	}()

	time.Sleep(
		100 * time.Millisecond,
	)

	cancel()

	select {
	case err :=
		<-done:

		if err == nil {
			t.Fatal(
				"expected canceled process to fail",
			)
		}

		if !errors.Is(
			ctx.Err(),
			context.Canceled,
		) {
			t.Fatalf(
				"expected canceled context, got %v",
				ctx.Err(),
			)
		}

	case <-time.After(
		5 * time.Second,
	):
		if cmd.Process != nil {
			_ =
				cmd.Process.Kill()
		}

		t.Fatal(
			"process did not terminate after cancellation",
		)
	}
}

func TestExternalCommandContextSetsWaitDelay(
	t *testing.T,
) {
	t.Parallel()

	cmd :=
		externalCommandContext(
			context.Background(),
			os.Args[0],
		)

	if cmd.WaitDelay !=
		externalProcessWaitDelay {
		t.Fatalf(
			"expected wait delay %s, got %s",
			externalProcessWaitDelay,
			cmd.WaitDelay,
		)
	}
}
