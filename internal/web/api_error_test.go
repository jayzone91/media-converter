package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
