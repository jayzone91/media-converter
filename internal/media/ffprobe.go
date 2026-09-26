package media

import (
	"context"
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
		CodecName string `json:"codec_name"`
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

func (p *FFProbe) Detect(ctx context.Context, path string) (Format, error) {
	cmd := exec.CommandContext(
		ctx,
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
		if ctx.Err() != nil {
			return Format{}, ctx.Err()
		}

		return Format{}, fmt.Errorf(
			"ffprobe failed: %w",
			err,
		)
	}

	var result probeResult

	if err := json.Unmarshal(output, &result); err != nil {
		return Format{}, fmt.Errorf(
			"failed to decode ffprobe output: %w",
			err,
		)
	}

	hasVideo := false
	hasAudio := false
	audioCodec := ""

	for _, stream := range result.Streams {
		switch stream.CodecType {
		case "video":
			hasVideo = true

		case "audio":
			hasAudio = true

			if audioCodec == "" {
				audioCodec = stream.CodecName
			}
		}
	}

	extension := strings.ToLower(
		filepath.Ext(path),
	)

	if hasVideo {
		switch extension {
		case ".mp4", ".m4v":
			return Formats["mp4"], nil

		case ".mov":
			return Formats["mov"], nil

		case ".mkv":
			return Formats["mkv"], nil

		case ".webm":
			return Formats["webm"], nil

		case ".avi":
			return Formats["avi"], nil

		case ".mpeg", ".mpg":
			return Formats["mpeg"], nil

		case ".wmv":
			return Formats["wmv"], nil

		case ".flv":
			return Formats["flv"], nil
		}

		switch {
		case hasProbeName(
			result.Format.FormatName,
			"matroska",
		):
			return Formats["mkv"], nil

		case hasProbeName(
			result.Format.FormatName,
			"webm",
		):
			return Formats["webm"], nil

		case hasProbeName(
			result.Format.FormatName,
			"avi",
		):
			return Formats["avi"], nil

		case hasProbeName(
			result.Format.FormatName,
			"mpeg",
		):
			return Formats["mpeg"], nil

		case hasProbeName(
			result.Format.FormatName,
			"asf",
		):
			return Formats["wmv"], nil

		case hasProbeName(
			result.Format.FormatName,
			"flv",
		):
			return Formats["flv"], nil
		}
	}

	if hasAudio && !hasVideo {
		switch extension {
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

		case ".wma":
			return Formats["wma"], nil
		}

		switch {
		case audioCodec == "opus" &&
			hasProbeName(
				result.Format.FormatName,
				"ogg",
			):
			return Formats["opus"], nil

		case hasProbeName(
			result.Format.FormatName,
			"mp3",
		):
			return Formats["mp3"], nil

		case hasProbeName(
			result.Format.FormatName,
			"wav",
		):
			return Formats["wav"], nil

		case hasProbeName(
			result.Format.FormatName,
			"flac",
		):
			return Formats["flac"], nil

		case hasProbeName(
			result.Format.FormatName,
			"aac",
		):
			return Formats["aac"], nil

		case hasProbeName(
			result.Format.FormatName,
			"ogg",
		):
			return Formats["ogg"], nil

		case hasProbeName(
			result.Format.FormatName,
			"asf",
		):
			return Formats["wma"], nil
		}
	}

	return Format{}, fmt.Errorf(
		"unsupported ffprobe format: %s",
		result.Format.FormatName,
	)
}

func hasProbeName(formatNames, name string) bool {
	for _, candidate := range strings.Split(
		formatNames,
		",",
	) {
		if candidate == name {
			return true
		}
	}

	return false
}
