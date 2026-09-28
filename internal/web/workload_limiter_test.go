package web

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWorkloadLimiterAllowsConfiguredConcurrency(
	t *testing.T,
) {
	t.Parallel()

	limiter :=
		newWorkloadLimiter()

	ctx :=
		context.Background()

	if err :=
		limiter.Acquire(
			ctx,
			workloadFFmpeg,
		); err != nil {
		t.Fatalf(
			"acquire first FFmpeg slot: %v",
			err,
		)
	}

	defer limiter.Release(
		workloadFFmpeg,
	)

	if err :=
		limiter.Acquire(
			ctx,
			workloadFFmpeg,
		); err != nil {
		t.Fatalf(
			"acquire second FFmpeg slot: %v",
			err,
		)
	}

	defer limiter.Release(
		workloadFFmpeg,
	)
}

func TestWorkloadLimiterBlocksWhenWorkloadIsFull(
	t *testing.T,
) {
	t.Parallel()

	limiter :=
		newWorkloadLimiter()

	ctx :=
		context.Background()

	if err :=
		limiter.Acquire(
			ctx,
			workloadGhostscript,
		); err != nil {
		t.Fatalf(
			"acquire Ghostscript slot: %v",
			err,
		)
	}

	defer limiter.Release(
		workloadGhostscript,
	)

	waitCtx, cancel :=
		context.WithTimeout(
			context.Background(),
			50*time.Millisecond,
		)

	defer cancel()

	err :=
		limiter.Acquire(
			waitCtx,
			workloadGhostscript,
		)

	if !errors.Is(
		err,
		context.DeadlineExceeded,
	) {
		t.Fatalf(
			"expected deadline exceeded, got %v",
			err,
		)
	}
}

func TestWorkloadLimiterKeepsWorkloadsIndependent(
	t *testing.T,
) {
	t.Parallel()

	limiter :=
		newWorkloadLimiter()

	ctx :=
		context.Background()

	if err :=
		limiter.Acquire(
			ctx,
			workloadGhostscript,
		); err != nil {
		t.Fatalf(
			"acquire Ghostscript slot: %v",
			err,
		)
	}

	defer limiter.Release(
		workloadGhostscript,
	)

	acquireCtx, cancel :=
		context.WithTimeout(
			context.Background(),
			100*time.Millisecond,
		)

	defer cancel()

	if err :=
		limiter.Acquire(
			acquireCtx,
			workloadLibreOffice,
		); err != nil {
		t.Fatalf(
			"LibreOffice should not be blocked by Ghostscript: %v",
			err,
		)
	}

	limiter.Release(
		workloadLibreOffice,
	)
}

func TestWorkloadLimiterUnblocksAfterRelease(
	t *testing.T,
) {
	t.Parallel()

	limiter :=
		newWorkloadLimiter()

	if err :=
		limiter.Acquire(
			context.Background(),
			workloadGhostscript,
		); err != nil {
		t.Fatalf(
			"acquire Ghostscript slot: %v",
			err,
		)
	}

	acquired :=
		make(
			chan error,
			1,
		)

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			time.Second,
		)

	defer cancel()

	go func() {
		err :=
			limiter.Acquire(
				ctx,
				workloadGhostscript,
			)

		acquired <- err
	}()

	select {
	case err :=
		<-acquired:
		t.Fatalf(
			"second acquire returned before release: %v",
			err,
		)

	case <-time.After(
		50 * time.Millisecond,
	):
	}

	limiter.Release(
		workloadGhostscript,
	)

	select {
	case err :=
		<-acquired:

		if err != nil {
			t.Fatalf(
				"acquire after release failed: %v",
				err,
			)
		}

		limiter.Release(
			workloadGhostscript,
		)

	case <-time.After(
		time.Second,
	):
		t.Fatal(
			"waiting acquire did not unblock after release",
		)
	}
}

func TestWorkloadLimiterRejectsUnknownWorkload(
	t *testing.T,
) {
	t.Parallel()

	limiter :=
		newWorkloadLimiter()

	err :=
		limiter.Acquire(
			context.Background(),
			workloadType(
				"unknown",
			),
		)

	if err == nil {
		t.Fatal(
			"expected unknown workload to be rejected",
		)
	}
}
