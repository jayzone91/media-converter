// Package converter provides media conversion backends
package converter

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type FFmpeg struct {
	binary string
}

func NewFFmpeg() (*FFmpeg, error) {
	binary, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg not found: %w", err)
	}

	return &FFmpeg{
		binary: binary,
	}, nil
}

func (c *FFmpeg) Convert(
	ctx context.Context,
	input string,
	output string,
) error {
	target :=
		strings.TrimPrefix(
			strings.ToLower(
				filepath.Ext(
					output,
				),
			),
			".",
		)

	args, err :=
		ffmpegArguments(
			input,
			output,
			target,
		)

	if err != nil {
		return err
	}

	cmd :=
		externalCommandContext(
			ctx,
			c.binary,
			args...,
		)

	return runExternalTool(
		ctx,
		"ffmpeg",
		cmd,
	)
}

func ffmpegArguments(input, output, target string) ([]string, error) {
	base := []string{
		"-y",
		"-i",
		input,
	}

	switch target {
	case "mp3":
		return append(
			base,
			"-vn",
			"-c:a",
			"libmp3lame",
			"-q:a",
			"2",
			output,
		), nil

	case "wav":
		return append(
			base,
			"-vn",
			"-c:a",
			"pcm_s16le",
			output,
		), nil

	case "flac":
		return append(
			base,
			"-vn",
			"-c:a",
			"flac",
			output,
		), nil

	case "ogg":
		return append(
			base,
			"-vn",
			"-c:a",
			"libvorbis",
			"-q:a",
			"6",
			output,
		), nil

	case "opus":
		return append(
			base,
			"-vn",
			"-c:a",
			"libopus",
			"-b:a",
			"128k",
			output,
		), nil

	case "aac":
		return append(
			base,
			"-vn",
			"-c:a",
			"aac",
			"-b:a",
			"192k",
			output,
		), nil

	case "m4a":
		return append(
			base,
			"-vn",
			"-c:a",
			"aac",
			"-b:a",
			"192k",
			"-movflags",
			"+faststart",
			output,
		), nil

	case "mp4":
		return append(
			base,
			"-c:v",
			"libx264",
			"-preset",
			"medium",
			"-crf",
			"23",
			"-pix_fmt",
			"yuv420p",
			"-c:a",
			"aac",
			"-b:a",
			"192k",
			"-movflags",
			"+faststart",
			output,
		), nil

	case "mkv":
		return append(
			base,
			"-c:v",
			"libx264",
			"-preset",
			"medium",
			"-crf",
			"23",
			"-c:a",
			"aac",
			"-b:a",
			"192k",
			output,
		), nil

	case "mov":
		return append(
			base,
			"-c:v",
			"libx264",
			"-preset",
			"medium",
			"-crf",
			"23",
			"-pix_fmt",
			"yuv420p",
			"-c:a",
			"aac",
			"-b:a",
			"192k",
			"-movflags",
			"+faststart",
			output,
		), nil

	case "webm":
		return append(
			base,
			"-c:v",
			"libvpx-vp9",
			"-crf",
			"32",
			"-b:v",
			"0",
			"-c:a",
			"libopus",
			"-b:a",
			"128k",
			output,
		), nil

	case "avi":
		return append(
			base,
			"-c:v",
			"mpeg4",
			"-q:v",
			"5",
			"-c:a",
			"libmp3lame",
			"-q:a",
			"4",
			output,
		), nil

	case "gif":
		return append(
			base,
			"-an",
			"-filter_complex",
			"fps=15,scale=1280:-2:force_original_aspect_ratio=decrease,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse",
			output,
		), nil

	default:
		return nil, fmt.Errorf(
			"unsupported ffmpeg target: %s",
			target,
		)
	}
}
