package netboxx

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Sentinel domain errors returned by the service layer.
var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrValidation   = errors.New("invalid argument")
)

// maxErrBody caps how much of an error response body is included in the error.
const maxErrBody = 2048

// apiError converts a non-2xx NetBox response into a domain error, keeping a
// snippet of the body (NetBox returns a JSON {"detail": ...} explanation).
func apiError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrBody))
	detail := strings.TrimSpace(string(body))
	if detail == "" {
		detail = resp.Status
	}

	var sentinel error
	switch resp.StatusCode {
	case http.StatusNotFound:
		sentinel = ErrNotFound
	case http.StatusUnauthorized:
		sentinel = ErrUnauthorized
	case http.StatusForbidden:
		sentinel = ErrForbidden
	default:
		return fmt.Errorf("netbox API error %d: %s", resp.StatusCode, detail)
	}
	return fmt.Errorf("%w: %s", sentinel, detail)
}
