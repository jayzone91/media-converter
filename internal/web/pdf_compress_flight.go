package web

import (
	"context"
	"sync"
)

type pdfCompressionFlight struct {
	done chan struct{}

	result pdfCompressionResult
	err    error
}

type pdfCompressionFlightGroup struct {
	mu sync.Mutex

	flights map[string]*pdfCompressionFlight
}

func newPDFCompressionFlightGroup() *pdfCompressionFlightGroup {
	return &pdfCompressionFlightGroup{
		flights: make(
			map[string]*pdfCompressionFlight,
		),
	}
}

func (g *pdfCompressionFlightGroup) Do(
	ctx context.Context,
	key string,
	fn func() (pdfCompressionResult, error),
) (pdfCompressionResult, error) {
	g.mu.Lock()

	if flight, ok :=
		g.flights[key]; ok {
		g.mu.Unlock()

		select {
		case <-flight.done:
			return flight.result,
				flight.err

		case <-ctx.Done():
			return pdfCompressionResult{},
				ctx.Err()
		}
	}

	flight :=
		&pdfCompressionFlight{
			done: make(
				chan struct{},
			),
		}

	g.flights[key] =
		flight

	g.mu.Unlock()

	result, err :=
		fn()

	g.mu.Lock()

	flight.result =
		result

	flight.err =
		err

	close(
		flight.done,
	)

	delete(
		g.flights,
		key,
	)

	g.mu.Unlock()

	return result, err
}
