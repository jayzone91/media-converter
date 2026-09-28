package web

import (
	"context"
	"fmt"
)

type workloadType string

const (
	workloadFFmpeg      workloadType = "ffmpeg"
	workloadImageMagick workloadType = "imagemagick"
	workloadGhostscript workloadType = "ghostscript"
	workloadChromium    workloadType = "chromium"
	workloadLibreOffice workloadType = "libreoffice"
	workloadQPDF        workloadType = "qpdf"
)

const (
	maxConcurrentFFmpeg      = 2
	maxConcurrentImageMagick = 2
	maxConcurrentGhostscript = 1
	maxConcurrentChromium    = 2
	maxConcurrentLibreOffice = 1
	maxConcurrentQPDF        = 2
)

type workloadLimiter struct {
	slots map[workloadType]chan struct{}
}

func newWorkloadLimiter() *workloadLimiter {
	return &workloadLimiter{
		slots: map[workloadType]chan struct{}{
			workloadFFmpeg: make(
				chan struct{},
				maxConcurrentFFmpeg,
			),

			workloadImageMagick: make(
				chan struct{},
				maxConcurrentImageMagick,
			),

			workloadGhostscript: make(
				chan struct{},
				maxConcurrentGhostscript,
			),

			workloadChromium: make(
				chan struct{},
				maxConcurrentChromium,
			),

			workloadLibreOffice: make(
				chan struct{},
				maxConcurrentLibreOffice,
			),

			workloadQPDF: make(
				chan struct{},
				maxConcurrentQPDF,
			),
		},
	}
}

func (l *workloadLimiter) Acquire(
	ctx context.Context,
	workload workloadType,
) error {
	slots, ok :=
		l.slots[workload]

	if !ok {
		return fmt.Errorf(
			"unknown workload %q",
			workload,
		)
	}

	queueCtx, cancel :=
		context.WithTimeout(
			ctx,
			conversionQueueTimeout,
		)

	defer cancel()

	select {
	case slots <- struct{}{}:
		return nil

	case <-queueCtx.Done():
		return queueCtx.Err()
	}
}

func (l *workloadLimiter) Release(
	workload workloadType,
) {
	slots, ok :=
		l.slots[workload]

	if !ok {
		return
	}

	<-slots
}
