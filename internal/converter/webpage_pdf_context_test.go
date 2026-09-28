package converter

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

func TestWebPDFBrowserContextFollowsRequestCancellation(
	t *testing.T,
) {
	t.Parallel()

	requestContext, cancelRequest :=
		context.WithCancel(
			context.Background(),
		)

	browser :=
		newWebPDFBrowserContext(
			requestContext,
			chromedp.ExecPath(
				os.Args[0],
			),
		)

	defer browser.Close()

	cancelRequest()

	select {
	case <-browser.Context.Done():

	case <-time.After(
		time.Second,
	):
		t.Fatal(
			"browser context was not canceled with request",
		)
	}

	if !errors.Is(
		browser.Context.Err(),
		context.Canceled,
	) {
		t.Fatalf(
			"expected canceled browser context, got %v",
			browser.Context.Err(),
		)
	}
}

func TestWebPDFBrowserContextCloseCancelsContext(
	t *testing.T,
) {
	t.Parallel()

	browser :=
		newWebPDFBrowserContext(
			context.Background(),
			chromedp.ExecPath(
				os.Args[0],
			),
		)

	browser.Close()

	select {
	case <-browser.Context.Done():

	case <-time.After(
		time.Second,
	):
		t.Fatal(
			"browser context was not canceled on close",
		)
	}
}

func TestWebPDFBrowserContextCloseIsIdempotent(
	t *testing.T,
) {
	t.Parallel()

	browser :=
		newWebPDFBrowserContext(
			context.Background(),
			chromedp.ExecPath(
				os.Args[0],
			),
		)

	browser.Close()
	browser.Close()

	if !errors.Is(
		browser.Context.Err(),
		context.Canceled,
	) {
		t.Fatalf(
			"expected canceled browser context, got %v",
			browser.Context.Err(),
		)
	}
}
