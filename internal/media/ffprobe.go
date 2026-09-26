package media

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type FFProbe struct {
	binary string
}

type probeResult struct {
	Format struct {
		FormatName string `json:"format_name"`
	} `json:"format"`
	Streams []struct {
		CodecType string `json:"codec_type"`
	} `json:"stremas"`
}

func NewFFProbe() (*FFProbe, error) {
	binary, err := exec.LookPath("ffprobe")
	if err != nil {
		return nil, fmt.Errorf("ffprobe not found: %w", err)
	}

	return &FFProbe{
		binary: binary,
	}, nil
}

func (p *FFProbe) Detect(path string) (Format, error) {
	cmd := exec.Command(
		p.binary,
		"-v",
		"error",
		"-show_format",
		"-show_streams",
		"-of",
		"json",
		path,
	)

	output, err := cmd.Output()
	if err != nil {
		return Format{}, fmt.Errorf("ffprobe failed: %w", err)
	}

	var result probeResult

	if err := json.Unmarshal(output, &result); err != nil {
		return Format{}, fmt.Errorf("failed to decode ffprobe output: %w", err)
	}

	for _, name := range strings.Split(result.Format.FormatName, ",") {
		if format, ok := FindByProbeName(name); ok {
			return format, nil
		}
	}

	return Format{}, fmt.Errorf("unsupported ffprobe format: %s", result.Format.FormatName)
}
