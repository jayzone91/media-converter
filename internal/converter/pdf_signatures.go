package converter

import (
	"errors"
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func HasPDFSignatures(
	path string,
) (bool, error) {
	results, err :=
		api.ValidateSignatures(
			path,
			true,
			nil,
		)

	if err == nil {
		return len(results) > 0,
			nil
	}

	if errors.Is(
		err,
		api.ErrNoSignatures,
	) {
		return false, nil
	}

	/*
		Eine vorhandene, aber beschädigte oder
		nicht vollständig validierbare Signatur
		darf nicht stillschweigend als
		"keine Signatur" behandelt werden.

		Im Zweifel wird die Optimierung daher
		abgebrochen.
	*/
	return false,
		fmt.Errorf(
			"inspect PDF signatures: %w",
			err,
		)
}
