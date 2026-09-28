package converter

import (
	"context"
	"sync"

	"github.com/chromedp/chromedp"
)

type webPDFBrowserContext struct {
	Context context.Context

	closeOnce sync.Once

	stopRequestCancellation func() bool

	runCancel       context.CancelFunc
	browserCancel   context.CancelFunc
	allocatorCancel context.CancelFunc
}

func newWebPDFBrowserContext(
	requestContext context.Context,
	options ...chromedp.ExecAllocatorOption,
) *webPDFBrowserContext {
	/*
		Der Chromium-Prozess gehört bewusst nicht direkt dem
		Request-Kontext.

		Ein abgebrochener Request beendet stattdessen den
		Chromium-Run-Kontext. Dadurch kann unser eigener Cleanup
		anschließend Browser und ExecAllocator kontrolliert
		aufräumen.
	*/
	allocatorContext, allocatorCancel :=
		chromedp.NewExecAllocator(
			context.Background(),
			options...,
		)

	browserContext, browserCancel :=
		chromedp.NewContext(
			allocatorContext,
		)

	runContext, runCancel :=
		context.WithCancel(
			browserContext,
		)

	browser :=
		&webPDFBrowserContext{
			Context: runContext,

			runCancel: runCancel,

			browserCancel: browserCancel,

			allocatorCancel: allocatorCancel,
		}

	browser.stopRequestCancellation =
		context.AfterFunc(
			requestContext,
			runCancel,
		)

	return browser
}

func (b *webPDFBrowserContext) Close() {
	b.closeOnce.Do(
		func() {
			if b.stopRequestCancellation != nil {
				b.stopRequestCancellation()
			}

			/*
				Zuerst laufende Aktionen abbrechen,
				dann Browser-Kontext schließen und zuletzt
				den Chromium-Allocator inklusive Prozess
				beenden.
			*/
			b.runCancel()
			b.browserCancel()
			b.allocatorCancel()
		},
	)
}
