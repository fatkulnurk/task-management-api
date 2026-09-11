package http

import (
	"encoding/json"
	"errors"
	"io"
	stdhttp "net/http"
	"taskmanagement/internal/application/errorcode"
)

// maxRequestBodyBytes is the biggest body we accept: 1 MiB.
const maxRequestBodyBytes = 1_048_576

// DecodeJSON reads JSON from the body into value.
// If the body is bad, it writes a 4xx answer and returns false.
// Then the handler can stop.
func DecodeJSON(responseWriter stdhttp.ResponseWriter, request *stdhttp.Request, value any) bool {
	// Stop a very big body. It can use too much memory.
	request.Body = stdhttp.MaxBytesReader(responseWriter, request.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(request.Body)
	err := decoder.Decode(value)
	if err != nil {
		// Three kinds of bad body: too big, empty, or broken JSON.
		var maxBytesError *stdhttp.MaxBytesError
		if errors.As(err, &maxBytesError) {
			JSONResponse4xx(responseWriter, stdhttp.StatusRequestEntityTooLarge, errorcode.PayloadTooLarge, "request body is too large")
			return false
		}
		if errors.Is(err, io.EOF) {
			// The body is empty.
			JSONResponse4xx(responseWriter, stdhttp.StatusBadRequest, errorcode.InvalidRequest, "request body is required")
			return false
		}
		JSONResponse4xx(responseWriter, stdhttp.StatusBadRequest, errorcode.InvalidJSON, "invalid JSON")
		return false
	}
	// Read again. If more JSON is here, the body is not valid.
	// For example: {...}{...}
	if err = decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		JSONResponse4xx(responseWriter, stdhttp.StatusBadRequest, errorcode.InvalidJSON, "invalid JSON")
		return false
	}
	return true
}
