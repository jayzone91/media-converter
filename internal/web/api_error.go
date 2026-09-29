package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

type apiErrorCode string

const (
	apiErrorInvalidRequest apiErrorCode = "invalid_request"

	apiErrorUploadExpired apiErrorCode = "upload_expired"

	apiErrorUnsupportedConversion apiErrorCode = "unsupported_conversion"

	apiErrorQueueTimeout apiErrorCode = "queue_timeout"

	apiErrorTimeout apiErrorCode = "timeout"

	apiErrorInternal apiErrorCode = "internal_error"
)

type apiErrorResponse struct {
	Code apiErrorCode `json:"code"`

	Message string `json:"message"`
}

func writeAPIError(
	w http.ResponseWriter,
	status int,
	code apiErrorCode,
	message string,
) {
	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	w.WriteHeader(
		status,
	)

	_ = json.NewEncoder(
		w,
	).Encode(
		apiErrorResponse{
			Code: code,

			Message: message,
		},
	)
}

func writeQueueAPIError(
	w http.ResponseWriter,
	err error,
) {
	if errors.Is(
		err,
		context.Canceled,
	) {
		return
	}

	if errors.Is(
		err,
		context.DeadlineExceeded,
	) {
		writeAPIError(
			w,
			http.StatusServiceUnavailable,
			apiErrorQueueTimeout,
			"Der Server ist momentan ausgelastet. Bitte später erneut versuchen.",
		)

		return
	}

	writeAPIError(
		w,
		http.StatusServiceUnavailable,
		apiErrorInternal,
		"Die Verarbeitung konnte nicht gestartet werden.",
	)
}

func writeTimeoutAPIError(
	w http.ResponseWriter,
	ctx context.Context,
	message string,
) bool {
	if errors.Is(
		ctx.Err(),
		context.Canceled,
	) {
		return true
	}

	if !errors.Is(
		ctx.Err(),
		context.DeadlineExceeded,
	) {
		return false
	}

	writeAPIError(
		w,
		http.StatusGatewayTimeout,
		apiErrorTimeout,
		message,
	)

	return true
}

func writeOperationContextAPIError(
	w http.ResponseWriter,
	ctx context.Context,
	err error,
	timeoutMessage string,
) bool {
	/*
		Der äußere Operations-Context entscheidet zuerst.

		Ist er abgelaufen, handelt es sich um den eigentlichen
		Verarbeitungs-Timeout und nicht um einen Queue-Timeout.
	*/
	if writeTimeoutAPIError(
		w,
		ctx,
		timeoutMessage,
	) {
		return true
	}

	/*
		Ein Workload-Limiter besitzt einen eigenen Queue-Context.

		Wenn dieser abläuft, ist ctx selbst weiterhin aktiv,
		während der zurückgegebene Fehler
		context.DeadlineExceeded enthält.
	*/
	if errors.Is(
		err,
		context.DeadlineExceeded,
	) {
		writeQueueAPIError(
			w,
			err,
		)

		return true
	}

	if errors.Is(
		err,
		context.Canceled,
	) {
		return true
	}

	return false
}
