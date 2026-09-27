package converter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type WebPDF struct {
	browserPath string
}

type WebPDFOptions struct {
	PaperSize       string
	RenderMode      string
	Landscape       bool
	PrintBackground bool
	Wait            time.Duration
}

func NewWebPDF() (*WebPDF, error) {
	browserPath, err :=
		findBrowserExecutable()

	if err != nil {
		return nil, err
	}

	return &WebPDF{
		browserPath: browserPath,
	}, nil
}

func (c *WebPDF) Render(
	ctx context.Context,
	url string,
	output string,
	options WebPDFOptions,
) error {
	paperWidth, paperHeight, err :=
		webPDFPaperDimensions(
			options.PaperSize,
		)

	if err != nil {
		return err
	}

	profile, err :=
		webPDFRenderProfileFor(
			options.RenderMode,
		)

	if err != nil {
		return err
	}

	profileDirectory, err :=
		os.MkdirTemp(
			"",
			"media-converter-chromium-*",
		)

	if err != nil {
		return fmt.Errorf(
			"create Chromium profile directory: %w",
			err,
		)
	}

	defer os.RemoveAll(
		profileDirectory,
	)

	allocatorOptions :=
		append(
			[]chromedp.ExecAllocatorOption{},
			chromedp.DefaultExecAllocatorOptions[:]...,
		)

	allocatorOptions =
		append(
			allocatorOptions,
			chromedp.ExecPath(
				c.browserPath,
			),
			chromedp.UserDataDir(
				profileDirectory,
			),
			chromedp.WindowSize(
				1440,
				1366,
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

	allocatorCtx, allocatorCancel :=
		chromedp.NewExecAllocator(
			ctx,
			allocatorOptions...,
		)
	defer allocatorCancel()

	browserCtx, browserCancel :=
		chromedp.NewContext(
			allocatorCtx,
		)
	defer browserCancel()

	var pdfData []byte

	actions :=
		webPDFProfileActions(
			profile,
		)

	actions = append(
		actions,
		chromedp.Navigate(
			url,
		),
	)

	if options.Wait > 0 {
		actions = append(
			actions,
			chromedp.Sleep(
				options.Wait,
			),
		)
	}

	scale :=
		webPDFPrintScale(
			profile,
			paperWidth,
			paperHeight,
			options.Landscape,
		)

	actions = append(
		actions,
		chromedp.ActionFunc(
			func(
				ctx context.Context,
			) error {
				data, _, err :=
					page.
						PrintToPDF().
						WithLandscape(
							options.Landscape,
						).
						WithPrintBackground(
							options.PrintBackground,
						).
						WithPaperWidth(
							paperWidth,
						).
						WithPaperHeight(
							paperHeight,
						).
						WithMarginTop(
							webPDFMarginInches,
						).
						WithMarginBottom(
							webPDFMarginInches,
						).
						WithMarginLeft(
							webPDFMarginInches,
						).
						WithMarginRight(
							webPDFMarginInches,
						).
						WithScale(
							scale,
						).
						WithPreferCSSPageSize(
							false,
						).
						Do(
							ctx,
						)

				if err != nil {
					return err
				}

				pdfData =
					data

				return nil
			},
		),
	)

	if err := chromedp.Run(
		browserCtx,
		actions...,
	); err != nil {
		return fmt.Errorf(
			"render webpage: %w",
			err,
		)
	}

	if len(pdfData) == 0 {
		return fmt.Errorf(
			"browser returned an empty PDF",
		)
	}

	if err := os.WriteFile(
		output,
		pdfData,
		0600,
	); err != nil {
		return fmt.Errorf(
			"write webpage PDF: %w",
			err,
		)
	}

	return nil
}

func webPDFPaperDimensions(
	paperSize string,
) (float64, float64, error) {
	switch strings.ToLower(
		paperSize,
	) {
	case "a4":
		return 8.2677165354,
			11.6929133858,
			nil

	case "letter":
		return 8.5,
			11,
			nil

	default:
		return 0,
			0,
			fmt.Errorf(
				"unsupported paper size: %s",
				paperSize,
			)
	}
}

func findBrowserExecutable() (
	string,
	error,
) {
	if override :=
		strings.TrimSpace(
			os.Getenv(
				"CHROME_BIN",
			),
		); override != "" {
		if path, ok :=
			resolveBrowserExecutable(
				override,
			); ok {
			return path,
				nil
		}
	}

	for _, candidate := range browserExecutableNames() {
		if path, err :=
			exec.LookPath(
				candidate,
			); err == nil {
			return path,
				nil
		}
	}

	for _, candidate := range browserExecutablePaths() {
		if info, err :=
			os.Stat(
				candidate,
			); err == nil &&
			!info.IsDir() {
			return candidate,
				nil
		}
	}

	return "",
		fmt.Errorf(
			"Chrome, Chromium or Edge not found",
		)
}

func resolveBrowserExecutable(
	value string,
) (string, bool) {
	if info, err :=
		os.Stat(
			value,
		); err == nil &&
		!info.IsDir() {
		return value,
			true
	}

	path, err :=
		exec.LookPath(
			value,
		)

	if err != nil {
		return "",
			false
	}

	return path,
		true
}

func browserExecutableNames() []string {
	if runtime.GOOS ==
		"windows" {
		return []string{
			"chrome.exe",
			"msedge.exe",
			"chromium.exe",
			"chrome",
			"msedge",
			"chromium",
		}
	}

	return []string{
		"chromium",
		"chromium-browser",
		"google-chrome",
		"google-chrome-stable",
		"microsoft-edge",
		"microsoft-edge-stable",
		"chrome",
	}
}

func browserExecutablePaths() []string {
	var paths []string

	add :=
		func(
			base string,
			parts ...string,
		) {
			if base == "" {
				return
			}

			paths = append(
				paths,
				filepath.Join(
					append(
						[]string{
							base,
						},
						parts...,
					)...,
				),
			)
		}

	if runtime.GOOS ==
		"windows" {
		for _, base := range []string{
			os.Getenv(
				"LOCALAPPDATA",
			),
			os.Getenv(
				"PROGRAMFILES",
			),
			os.Getenv(
				"PROGRAMFILES(X86)",
			),
		} {
			add(
				base,
				"Google",
				"Chrome",
				"Application",
				"chrome.exe",
			)

			add(
				base,
				"Microsoft",
				"Edge",
				"Application",
				"msedge.exe",
			)
		}
	}

	if runtime.GOOS ==
		"darwin" {
		paths = append(
			paths,
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		)
	}

	return paths
}
