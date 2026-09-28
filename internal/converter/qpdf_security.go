package converter

import (
	"context"
	"fmt"
	"os"
	"strings"
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

	responseFile, err :=
		createQPDFEncryptionResponseFile(
			input,
			userPassword,
			ownerPassword,
			output,
		)

	if err != nil {
		return err
	}

	defer func() {
		_ = os.Remove(
			responseFile,
		)
	}()

	return q.run(
		ctx,
		"encrypt",
		[]string{
			"@" + responseFile,
		},
	)
}

func (q *QPDF) Decrypt(
	ctx context.Context,
	input string,
	password string,
	output string,
) error {
	passwordFile, err :=
		createQPDFPasswordFile(
			password,
		)

	if err != nil {
		return err
	}

	defer func() {
		_ = os.Remove(
			passwordFile,
		)
	}()

	args := []string{
		"--warning-exit-0",

		"--password-file=" +
			passwordFile,

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

func createQPDFEncryptionResponseFile(
	input string,
	userPassword string,
	ownerPassword string,
	output string,
) (string, error) {
	if err :=
		validateQPDFResponseValue(
			userPassword,
		); err != nil {
		return "",
			fmt.Errorf(
				"invalid user password: %w",
				err,
			)
	}

	if err :=
		validateQPDFResponseValue(
			ownerPassword,
		); err != nil {
		return "",
			fmt.Errorf(
				"invalid owner password: %w",
				err,
			)
	}

	content :=
		strings.Join(
			[]string{
				"--warning-exit-0",

				"--stream-data=preserve",

				"--password-mode=unicode",

				"--encrypt",

				"--user-password=" +
					userPassword,

				"--owner-password=" +
					ownerPassword,

				"--bits=256",

				"--",

				input,

				output,
			},
			"\n",
		)

	path, err :=
		writeQPDFSecretFile(
			"media-converter-qpdf-encrypt-*",
			content,
		)

	if err != nil {
		return "",
			fmt.Errorf(
				"create qpdf encryption response file: %w",
				err,
			)
	}

	return path, nil
}

func createQPDFPasswordFile(
	password string,
) (string, error) {
	if err :=
		validateQPDFResponseValue(
			password,
		); err != nil {
		return "",
			fmt.Errorf(
				"invalid qpdf password: %w",
				err,
			)
	}

	path, err :=
		writeQPDFSecretFile(
			"media-converter-qpdf-password-*",
			password,
		)

	if err != nil {
		return "",
			fmt.Errorf(
				"create qpdf password file: %w",
				err,
			)
	}

	return path, nil
}

func writeQPDFSecretFile(
	pattern string,
	content string,
) (string, error) {
	file, err :=
		os.CreateTemp(
			"",
			pattern,
		)

	if err != nil {
		return "", err
	}

	path :=
		file.Name()

	remove :=
		true

	defer func() {
		_ = file.Close()

		if remove {
			_ = os.Remove(
				path,
			)
		}
	}()

	if err :=
		file.Chmod(
			0600,
		); err != nil {
		return "",
			fmt.Errorf(
				"restrict secret file permissions: %w",
				err,
			)
	}

	if _, err :=
		file.WriteString(
			content,
		); err != nil {
		return "",
			fmt.Errorf(
				"write secret file: %w",
				err,
			)
	}

	if err :=
		file.Close(); err != nil {
		return "",
			fmt.Errorf(
				"close secret file: %w",
				err,
			)
	}

	remove = false

	return path, nil
}

func validateQPDFResponseValue(
	value string,
) error {
	if strings.ContainsAny(
		value,
		"\r\n\x00",
	) {
		return fmt.Errorf(
			"value contains unsupported control characters",
		)
	}

	return nil
}
