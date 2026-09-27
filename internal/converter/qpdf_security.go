package converter

import (
	"context"
	"fmt"
)

func (q *QPDF) Encrypt(
	ctx context.Context,
	input string,
	userPassword string,
	ownerPassword string,
	output string,
) error {
	if userPassword == "" {
		return fmt.Errorf(
			"user password must not be empty",
		)
	}

	if ownerPassword == "" {
		return fmt.Errorf(
			"owner password must not be empty",
		)
	}

	args := []string{
		"--warning-exit-0",
		"--stream-data=preserve",
		"--encrypt",
		userPassword,
		ownerPassword,
		"256",
		"--",
		input,
		output,
	}

	return q.run(
		ctx,
		"encrypt",
		args,
	)
}

func (q *QPDF) Decrypt(
	ctx context.Context,
	input string,
	password string,
	output string,
) error {
	args := []string{
		"--warning-exit-0",
		"--password=" + password,
		"--decrypt",
		"--stream-data=preserve",
		input,
		output,
	}

	return q.run(
		ctx,
		"decrypt",
		args,
	)
}
