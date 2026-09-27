package converter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chromedp/chromedp"
)

const (
	markdownImageViewportWidth = 1200
	markdownImageMaxHeight     = 16000
)

func MarkdownToImage(
	ctx context.Context,
	webPDF *WebPDF,
	imageMagick *ImageMagick,
	inputPath string,
	outputPath string,
	target string,
) error {
	tempDir := filepath.Dir(outputPath)

	htmlPath := filepath.Join(
		tempDir,
		"markdown-image-render.html",
	)

	pngPath := filepath.Join(
		tempDir,
		"markdown-image-render.png",
	)

	if err := MarkdownToHTML(
		inputPath,
		htmlPath,
	); err != nil {
		return err
	}

	defer os.Remove(htmlPath)

	if err := renderMarkdownScreenshot(
		ctx,
		webPDF,
		htmlPath,
		pngPath,
	); err != nil {
		return err
	}

	if target == "png" {
		if err := os.Rename(
			pngPath,
			outputPath,
		); err != nil {
			return fmt.Errorf(
				"move markdown PNG: %w",
				err,
			)
		}

		return nil
	}

	defer os.Remove(pngPath)

	if target != "jpeg" && target != "webp" {
		return fmt.Errorf(
			"unsupported markdown image format: %s",
			target,
		)
	}

	if err := imageMagick.Convert(
		ctx,
		pngPath,
		outputPath,
	); err != nil {
		return fmt.Errorf(
			"convert markdown image: %w",
			err,
		)
	}

	return nil
}

func renderMarkdownScreenshot(
	ctx context.Context,
	webPDF *WebPDF,
	htmlPath string,
	outputPath string,
) error {
	fileURL, err := localFileURL(htmlPath)
	if err != nil {
		return err
	}

	profileDirectory, err := os.MkdirTemp(
		"",
		"media-converter-markdown-image-*",
	)
	if err != nil {
		return fmt.Errorf(
			"create Chromium profile directory: %w",
			err,
		)
	}
	defer os.RemoveAll(profileDirectory)

	allocatorOptions := append(
		[]chromedp.ExecAllocatorOption{},
		chromedp.DefaultExecAllocatorOptions[:]...,
	)

	allocatorOptions = append(
		allocatorOptions,
		chromedp.ExecPath(
			webPDF.browserPath,
		),
		chromedp.UserDataDir(
			profileDirectory,
		),
		chromedp.WindowSize(
			markdownImageViewportWidth,
			900,
		),
		chromedp.Flag(
			"disable-gpu",
			true,
		),
		chromedp.Flag(
			"disable-dev-shm-usage",
			true,
		),
		chromedp.Flag(
			"no-first-run",
			true,
		),
		chromedp.Flag(
			"no-default-browser-check",
			true,
		),
	)

	allocatorCtx, allocatorCancel := chromedp.NewExecAllocator(
		ctx,
		allocatorOptions...,
	)
	defer allocatorCancel()

	browserCtx, browserCancel := chromedp.NewContext(
		allocatorCtx,
	)
	defer browserCancel()

	var (
		height     int
		screenshot []byte
	)

	if err := chromedp.Run(
		browserCtx,
		chromedp.Navigate(
			fileURL,
		),
		chromedp.WaitReady(
			"body",
			chromedp.ByQuery,
		),
		chromedp.Evaluate(
			`Math.max(
				document.body.scrollHeight,
				document.body.offsetHeight,
				document.documentElement.clientHeight,
				document.documentElement.scrollHeight,
				document.documentElement.offsetHeight
			)`,
			&height,
		),
	); err != nil {
		return fmt.Errorf(
			"measure markdown document: %w",
			err,
		)
	}

	if height <= 0 {
		return fmt.Errorf(
			"markdown document has invalid render height",
		)
	}

	if height > markdownImageMaxHeight {
		return fmt.Errorf(
			"markdown document exceeds maximum image height of %d px",
			markdownImageMaxHeight,
		)
	}

	if err := chromedp.Run(
		browserCtx,
		chromedp.EmulateViewport(
			markdownImageViewportWidth,
			int64(height),
		),
		chromedp.FullScreenshot(
			&screenshot,
			100,
		),
	); err != nil {
		return fmt.Errorf(
			"capture markdown image: %w",
			err,
		)
	}

	if len(screenshot) == 0 {
		return fmt.Errorf(
			"browser returned an empty markdown image",
		)
	}

	if err := os.WriteFile(
		outputPath,
		screenshot,
		0o600,
	); err != nil {
		return fmt.Errorf(
			"write markdown image: %w",
			err,
		)
	}

	return nil
}
