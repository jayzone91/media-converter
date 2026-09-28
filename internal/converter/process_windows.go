//go:build windows

package converter

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

const createNewProcessGroup = 0x00000200

func configureProcessTree(
	cmd *exec.Cmd,
) {
	cmd.SysProcAttr =
		&syscall.SysProcAttr{
			CreationFlags: createNewProcessGroup,
		}
}

func terminateProcessTree(
	cmd *exec.Cmd,
) error {
	if cmd.Process == nil {
		return nil
	}

	pid :=
		strconv.Itoa(
			cmd.Process.Pid,
		)

	/*
		taskkill /T beendet auch alle untergeordneten Prozesse.
		Das ist für Programme wie LibreOffice relevant, die
		weitere Prozesse starten können.
	*/
	kill :=
		exec.Command(
			"taskkill",
			"/PID",
			pid,
			"/T",
			"/F",
		)

	if err :=
		kill.Run(); err == nil {
		return nil
	}

	/*
		Fallback: zumindest den direkten Prozess beenden,
		falls taskkill aus irgendeinem Grund nicht verfügbar ist.
	*/
	err :=
		cmd.Process.Kill()

	if err == nil ||
		errors.Is(
			err,
			os.ErrProcessDone,
		) {
		return nil
	}

	return fmt.Errorf(
		"terminate process tree: %w",
		err,
	)
}
