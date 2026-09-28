package web

import (
	"encoding/json"
	"net/http"
)

type apiErrorCode string

const (
	apiErrorInvalidRequest apiErrorCode = "invalid_request"

	apiErrorUploadExpired apiErrorCode = "upload_expired"

	apiErrorUnsupportedConversion apiErrorCode = "unsupported_conversion"

	apiErrorQueueTimeout apiErrorCode = "queue_timeout"

	apiErrorConversionTimeout apiErrorCode = "conversion_timeout"

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
