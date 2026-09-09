package http

import (
	"encoding/json"
	"errors"
	"io"
	stdhttp "net/http"
	"taskmanagement/internal/application/errorcode"
)

const maxRequestBodyBytes = 1 << 20

func DecodeJSON(responseWriter stdhttp.ResponseWriter, request *stdhttp.Request, value any) bool {
	request.Body = stdhttp.MaxBytesReader(responseWriter, request.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(request.Body)
	err := decoder.Decode(value)
	if err != nil {
		var maxBytesError *stdhttp.MaxBytesError
		if errors.As(err, &maxBytesError) {
			JSONResponse4xx(responseWriter, stdhttp.StatusRequestEntityTooLarge, errorcode.PayloadTooLarge, "request body is too large")
			return false
		}
		if errors.Is(err, io.EOF) {
			JSONResponse4xx(responseWriter, stdhttp.StatusBadRequest, errorcode.InvalidRequest, "request body is required")
			return false
		}
		JSONResponse4xx(responseWriter, stdhttp.StatusBadRequest, errorcode.InvalidJSON, "invalid JSON")
		return false
	}
	if err = decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		JSONResponse4xx(responseWriter, stdhttp.StatusBadRequest, errorcode.InvalidJSON, "invalid JSON")
		return false
	}
	return true
}
