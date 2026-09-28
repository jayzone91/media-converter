package converter

import (
	"context"
	"os/exec"
	"time"
)

const externalProcessWaitDelay = 5 * time.Second

func externalCommandContext(
	ctx context.Context,
	name string,
	args ...string,
) *exec.Cmd {
	cmd :=
		exec.CommandContext(
			ctx,
			name,
			args...,
		)

	configureProcessTree(
		cmd,
	)

	cmd.Cancel =
		func() error {
			return terminateProcessTree(
				cmd,
			)
		}

	/*
		Verhindert, dass Wait unbegrenzt auf offene Pipes
		eines bereits beendeten Prozesses wartet.
	*/
	cmd.WaitDelay =
		externalProcessWaitDelay

	return cmd
}
