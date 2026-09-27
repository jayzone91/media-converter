package web

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jayzone91/media-converter/internal/converter"
	"github.com/jayzone91/media-converter/internal/media"
)

func (s *Server) convertFile(
	ctx context.Context,
	format media.Format,
	target string,
	inputPath string,
	outputDir string,
	filename string,
) (string, string, error) {
	baseName := strings.TrimSuffix(
		filepath.Base(filename),
		filepath.Ext(filename),
	)

	outputName := baseName + "." + target

	if format.Category == media.CategoryPDF &&
		(target == "png" || target == "jpeg") {
		outputName = baseName + "-" + target + ".zip"
	}

	outputPath := filepath.Join(
		outputDir,
		outputName,
	)

	if err := os.MkdirAll(
		outputDir,
		0o700,
	); err != nil {
		return "", "",
			fmt.Errorf(
				"create output directory: %w",
				err,
			)
	}

	var err error

	switch format.Category {
	case media.CategoryImage:
		switch {
		case format.ID == "gif" &&
			(target == "mp4" || target == "webm"):
			err = s.ffmpeg.Convert(
				ctx,
				inputPath,
				outputPath,
			)

		default:
			err = s.imageMagick.Convert(
				ctx,
				inputPath,
				outputPath,
			)
		}

	case media.CategoryAudio, media.CategoryVideo:
		err = s.ffmpeg.Convert(
			ctx,
			inputPath,
			outputPath,
		)

	case media.CategoryDocument:
		err = s.libreOffice.Convert(
			ctx,
			inputPath,
			outputPath,
		)

	case media.CategoryMarkdown:
		switch target {
		case "html":
			err = converter.MarkdownToHTML(
				inputPath,
				outputPath,
			)

		case "pdf":
			err = converter.MarkdownToPDF(
				ctx,
				s.webPDF,
				inputPath,
				outputPath,
			)

		case "png", "jpeg", "webp":
			err = converter.MarkdownToImage(
				ctx,
				s.webPDF,
				s.imageMagick,
				inputPath,
				outputPath,
				target,
			)

		default:
			return "", "",
				fmt.Errorf(
					"unsupported Markdown conversion",
				)
		}

	case media.CategoryPDF:
		switch target {
		case "docx":
			err = s.pdf.ConvertToDOCX(
				ctx,
				inputPath,
				outputPath,
			)

		case "png", "jpeg":
			err = s.pdf.ConvertToImages(
				ctx,
				inputPath,
				outputPath,
				target,
			)

		default:
			return "", "",
				fmt.Errorf(
					"unsupported PDF conversion",
				)
		}

	default:
		return "", "",
			fmt.Errorf(
				"converter not implemented for this media type",
			)
	}

	if err != nil {
		return "", "", err
	}

	return outputPath,
		outputName,
		nil
}
