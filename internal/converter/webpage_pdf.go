package converter

import (
	"context"
	"fmt"
	"os"
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

	RequestValidator WebPDFRequestValidator
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

	if options.RequestValidator != nil {
		if err :=
			options.RequestValidator(
				ctx,
				url,
			); err != nil {
			return fmt.Errorf(
				"initial webpage request rejected: %w",
				err,
			)
		}
	}

	proxy, err :=
		startWebPDFProxy(
			ctx,
			options.RequestValidator,
		)

	if err != nil {
		return fmt.Errorf(
			"prepare webpage network proxy: %w",
			err,
		)
	}

	if proxy != nil {
		defer proxy.Close()
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

	if proxy != nil {
		allocatorOptions =
			append(
				allocatorOptions,

				chromedp.Flag(
					"proxy-server",
					proxy.URL(),
				),

				chromedp.Flag(
					"proxy-bypass-list",
					"<-loopback>",
				),
			)
	}

	browser :=
		newWebPDFBrowserContext(
			ctx,
			allocatorOptions...,
		)

	/*
		Dieser Cleanup läuft vor dem Entfernen des
		Profilverzeichnisses, weil sein defer später
		registriert wurde.
	*/
	defer browser.Close()

	blockedRequests :=
		make(
			chan error,
			1,
		)

	var pdfData []byte

	actions :=
		[]chromedp.Action{
			setupWebPDFRequestGuard(
				browser.Context,
				options.RequestValidator,
				blockedRequests,
			),
		}

	actions = append(
		actions,
		webPDFProfileActions(
			profile,
		)...,
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

	runErr :=
		chromedp.Run(
			browser.Context,
			actions...,
		)

	select {
	case blockedErr :=
		<-blockedRequests:

		return blockedErr

	default:
	}

	if runErr != nil {
		return newExternalToolError(
			ctx,
			"chromium",
			runErr,
			"",
		)
	}

	if len(pdfData) == 0 {
		return fmt.Errorf(
			"browser returned an empty PDF",
		)
	}

	if err :=
		os.WriteFile(
			output,
			pdfData,
			0o600,
		); err != nil {
		return fmt.Errorf(
			"write webpage PDF: %w",
			err,
		)
	}

	return nil
}
