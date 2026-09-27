package converter

import (
	"context"
	"fmt"
	"net/url"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

type WebPDFRequestValidator func(
	context.Context,
	string,
) error

func setupWebPDFRequestGuard(
	ctx context.Context,
	validator WebPDFRequestValidator,
	blocked chan<- error,
) chromedp.Action {
	if validator == nil {
		return chromedp.ActionFunc(
			func(context.Context) error {
				return nil
			},
		)
	}

	chromedp.ListenTarget(
		ctx,
		func(event any) {
			paused, ok :=
				event.(*fetch.EventRequestPaused)
			if !ok {
				return
			}

			go handleWebPDFPausedRequest(
				ctx,
				validator,
				blocked,
				paused,
			)
		},
	)

	return fetch.Enable()
}

func handleWebPDFPausedRequest(
	ctx context.Context,
	validator WebPDFRequestValidator,
	blocked chan<- error,
	event *fetch.EventRequestPaused,
) {
	requestURL := event.Request.URL

	parsed, err := url.Parse(
		requestURL,
	)
	if err != nil {
		reportBlockedWebPDFRequest(
			blocked,
			fmt.Errorf(
				"parse browser request URL: %w",
				err,
			),
		)

		failWebPDFRequest(
			ctx,
			event.RequestID,
		)

		return
	}

	switch parsed.Scheme {
	case "http", "https":
		if err := validator(
			ctx,
			requestURL,
		); err != nil {
			reportBlockedWebPDFRequest(
				blocked,
				fmt.Errorf(
					"blocked browser request to %s: %w",
					parsed.Hostname(),
					err,
				),
			)

			failWebPDFRequest(
				ctx,
				event.RequestID,
			)

			return
		}

	case "data", "blob", "about":
		// Lokale Browser-Ressourcen sind erlaubt.

	default:
		reportBlockedWebPDFRequest(
			blocked,
			fmt.Errorf(
				"blocked browser request scheme %q",
				parsed.Scheme,
			),
		)

		failWebPDFRequest(
			ctx,
			event.RequestID,
		)

		return
	}

	_ = chromedp.Run(
		ctx,
		fetch.ContinueRequest(
			event.RequestID,
		),
	)
}

func failWebPDFRequest(
	ctx context.Context,
	requestID fetch.RequestID,
) {
	_ = chromedp.Run(
		ctx,
		fetch.FailRequest(
			requestID,
			network.ErrorReasonBlockedByClient,
		),
	)
}

func reportBlockedWebPDFRequest(
	blocked chan<- error,
	err error,
) {
	select {
	case blocked <- err:
	default:
	}
}
