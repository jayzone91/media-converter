package converter

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type LibreOffice struct {
	binary string
}

func NewLibreOffice() (*LibreOffice, error) {
	binary, err := exec.LookPath("soffice")
	if err != nil {
		binary, err = exec.LookPath("libreoffice")
		if err != nil {
			return nil, fmt.Errorf("libreoffice not found: %w", err)
		}
	}

	return &LibreOffice{
		binary: binary,
	}, nil
}

func (c *LibreOffice) Convert(ctx context.Context, input, output string) error {
	target := strings.TrimPrefix(strings.ToLower(filepath.Ext(output)), ".")

	outputDir := filepath.Dir(output)

	cmd := externalCommandContext(
		ctx,
		c.binary,
		"--headless",
		"--convert-to",
		target,
		"--outdir",
		outputDir,
		input,
	)

	result, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("libreoffice failed: %w: %s", err, string(result))
	}

	return nil
}
