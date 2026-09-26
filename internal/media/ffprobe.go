package media

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
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
	} `json:"streams"`
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

	hasVideo := false
	hasAudio := false

	for _, stream := range result.Streams {
		switch stream.CodecType {
		case "video":
			hasVideo = true

		case "audio":
			hasAudio = true
		}
	}

	names := strings.Split(result.Format.FormatName, ",")

	if hasVideo {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".mov":
			return Formats["mov"], nil
		case ".mp4", ".m4v":
			return Formats["mp4"], nil
		}

		for _, name := range names {
			switch name {
			case "matroska":
				return Formats["mkv"], nil
			case "webm":
				return Formats["webm"], nil
			case "avi":
				return Formats["avi"], nil
			case "mpeg":
				return Formats["mpeg"], nil
			}
		}
	}

	if hasAudio && !hasVideo {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".m4a":
			return Formats["m4a"], nil
		case ".aac":
			return Formats["aac"], nil
		case ".mp3":
			return Formats["mp3"], nil
		case ".wav":
			return Formats["wav"], nil
		case ".flac":
			return Formats["flac"], nil
		case ".ogg":
			return Formats["ogg"], nil
		case ".opus":
			return Formats["opus"], nil
		}

		for _, name := range names {
			switch name {
			case "mp3":
				return Formats["mp3"], nil
			case "wav":
				return Formats["wav"], nil
			case "flac":
				return Formats["flac"], nil
			case "aac":
				return Formats["aac"], nil
			case "ogg":
				return Formats["ogg"], nil
			}
		}
	}

	return Format{}, fmt.Errorf("unsupported ffprobe format: %s", result.Format.FormatName)
}
