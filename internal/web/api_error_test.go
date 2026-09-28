package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWriteAPIError(
	t *testing.T,
) {
	t.Parallel()

	recorder :=
		httptest.NewRecorder()

	writeAPIError(
		recorder,
		http.StatusBadRequest,
		apiErrorInvalidRequest,
		"Ungültige Anfrage.",
	)

	response :=
		recorder.Result()

	defer response.Body.Close()

	if response.StatusCode !=
		http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			response.StatusCode,
		)
	}

	contentType :=
		response.Header.Get(
			"Content-Type",
		)

	if contentType !=
		"application/json; charset=utf-8" {
		t.Fatalf(
			"unexpected content type %q",
			contentType,
		)
	}

	body :=
		decodeAPIErrorResponse(
			t,
			response,
		)

	if body.Code !=
		apiErrorInvalidRequest {
		t.Fatalf(
			"expected code %q, got %q",
			apiErrorInvalidRequest,
			body.Code,
		)
	}

	if body.Message !=
		"Ungültige Anfrage." {
		t.Fatalf(
			"unexpected message %q",
			body.Message,
		)
	}
}

func TestWriteQueueAPIErrorTimeout(
	t *testing.T,
) {
	t.Parallel()

	recorder :=
		httptest.NewRecorder()

	writeQueueAPIError(
		recorder,
		context.DeadlineExceeded,
	)

	response :=
		recorder.Result()

	defer response.Body.Close()

	if response.StatusCode !=
		http.StatusServiceUnavailable {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusServiceUnavailable,
			response.StatusCode,
		)
	}

	body :=
		decodeAPIErrorResponse(
			t,
			response,
		)

	if body.Code !=
		apiErrorQueueTimeout {
		t.Fatalf(
			"expected code %q, got %q",
			apiErrorQueueTimeout,
			body.Code,
		)
	}
}

func TestWriteQueueAPIErrorCancellationWritesNothing(
	t *testing.T,
) {
	t.Parallel()

	recorder :=
		httptest.NewRecorder()

	writeQueueAPIError(
		recorder,
		context.Canceled,
	)

	if recorder.Body.Len() != 0 {
		t.Fatalf(
			"expected empty response, got %q",
			recorder.Body.String(),
		)
	}
}

func TestWriteTimeoutAPIError(
	t *testing.T,
) {
	t.Parallel()

	ctx, cancel :=
		context.WithDeadline(
			context.Background(),
			time.Now().
				Add(
					-time.Second,
				),
		)

	defer cancel()

	recorder :=
		httptest.NewRecorder()

	if !writeTimeoutAPIError(
		recorder,
		ctx,
		"Vorgang dauerte zu lange.",
	) {
		t.Fatal(
			"expected timeout to be handled",
		)
	}

	response :=
		recorder.Result()

	defer response.Body.Close()

	if response.StatusCode !=
		http.StatusGatewayTimeout {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusGatewayTimeout,
			response.StatusCode,
		)
	}

	body :=
		decodeAPIErrorResponse(
			t,
			response,
		)

	if body.Code !=
		apiErrorTimeout {
		t.Fatalf(
			"expected code %q, got %q",
			apiErrorTimeout,
			body.Code,
		)
	}
}

func TestWriteTimeoutAPIErrorIgnoresActiveContext(
	t *testing.T,
) {
	t.Parallel()

	recorder :=
		httptest.NewRecorder()

	if writeTimeoutAPIError(
		recorder,
		context.Background(),
		"unused",
	) {
		t.Fatal(
			"expected active context to remain unhandled",
		)
	}
}

func decodeAPIErrorResponse(
	t *testing.T,
	response *http.Response,
) apiErrorResponse {
	t.Helper()

	var body apiErrorResponse

	if err :=
		json.NewDecoder(
			response.Body,
		).Decode(
			&body,
		); err != nil {
		t.Fatalf(
			"decode API error: %v",
			err,
		)
	}

	return body
}
