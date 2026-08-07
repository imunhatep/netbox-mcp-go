package netboxx

import (
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// newLoggingHTTPClient builds the HTTP client used for every NetBox API call.
// Its transport logs each request/response, split by verbosity:
//
//   - info:  one line per response with status, byte size and elapsed time —
//     enough to spot a slow or oversized API response feeding the model.
//   - debug: adds a line per outgoing request (method + URL).
//   - trace: adds the request headers (with credentials redacted).
func newLoggingHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: &loggingTransport{base: http.DefaultTransport},
	}
}

type loggingTransport struct{ base http.RoundTripper }

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	log.Debug().Str("http_method", req.Method).Str("url", req.URL.String()).Msg("[netbox] request")
	if e := log.Trace(); e.Enabled() {
		e.Str("http_method", req.Method).Str("url", req.URL.String()).
			Interface("headers", redactHeaders(req.Header)).Msg("[netbox] request headers")
	}

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		log.Error().Err(err).Str("http_method", req.Method).Str("url", req.URL.String()).
			Dur("elapsed", time.Since(start)).Msg("[netbox] request failed")
		return nil, err
	}

	// Measure the true payload size as it is read, so the summary reflects
	// what actually crossed the wire. Logged once the body is closed.
	resp.Body = &countingBody{
		ReadCloser: resp.Body,
		start:      start,
		method:     req.Method,
		url:        req.URL.String(),
		status:     resp.StatusCode,
	}
	return resp, nil
}

type countingBody struct {
	io.ReadCloser
	start  time.Time
	method string
	url    string
	status int
	n      int64
	logged bool
}

func (b *countingBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.n += int64(n)
	return n, err
}

func (b *countingBody) Close() error {
	err := b.ReadCloser.Close()
	if !b.logged {
		b.logged = true
		log.Info().Str("http_method", b.method).Str("url", b.url).
			Int("status", b.status).Int64("resp_bytes", b.n).
			Dur("elapsed", time.Since(b.start)).Msg("[netbox] response")
	}
	return err
}

// redactHeaders returns a copy of h with credential-bearing headers masked, so
// trace logging never leaks the NetBox token.
func redactHeaders(h http.Header) http.Header {
	out := h.Clone()
	if out == nil {
		return nil
	}
	for _, k := range []string{"Authorization"} {
		if _, ok := out[k]; ok {
			out.Set(k, "REDACTED")
		}
	}
	return out
}
