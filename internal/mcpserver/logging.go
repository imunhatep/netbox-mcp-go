package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog/log"
)

// mcpLogger logs every MCP request and response. Verbosity is split so an
// operator can dial in exactly as much as they need:
//
//   - info:  one line per response with the method, elapsed time and content
//     size — enough to spot a tool returning a huge payload to the model.
//   - debug: adds a line per incoming request (method + id).
//   - trace: adds the full request params and full response content.
//
// It pairs OnBeforeAny with OnSuccess/OnError (the mcp-go pairing contract) to
// measure per-request latency, keyed by JSON-RPC request id.
type mcpLogger struct {
	starts sync.Map // request id (any) -> time.Time
}

// hooks builds the server hook registry backed by this logger.
func (l *mcpLogger) hooks() *server.Hooks {
	h := &server.Hooks{}
	h.AddBeforeAny(l.onBefore)
	h.AddOnSuccess(l.onSuccess)
	h.AddOnError(l.onError)
	return h
}

func (l *mcpLogger) onBefore(_ context.Context, id any, method mcp.MCPMethod, message any) {
	l.starts.Store(id, time.Now())
	log.Debug().Str("mcp_method", string(method)).Interface("id", id).Msg("[mcp] request")
	if e := log.Trace(); e.Enabled() {
		e.Str("mcp_method", string(method)).Interface("id", id).
			Interface("params", message).Msg("[mcp] request params")
	}
}

func (l *mcpLogger) onSuccess(_ context.Context, id any, method mcp.MCPMethod, message, result any) {
	ev := log.Info().Str("mcp_method", string(method)).Interface("id", id).
		Dur("elapsed", l.elapsed(id)).Int("resp_bytes", contentSize(result))
	if req, ok := message.(*mcp.CallToolRequest); ok {
		ev.Str("tool", req.Params.Name)
	}
	ev.Msg("[mcp] response")

	if e := log.Trace(); e.Enabled() {
		if b, err := json.Marshal(result); err == nil {
			e.Str("mcp_method", string(method)).Interface("id", id).
				RawJSON("content", b).Msg("[mcp] response content")
		}
	}
}

func (l *mcpLogger) onError(_ context.Context, id any, method mcp.MCPMethod, _ any, err error) {
	log.Error().Err(err).Str("mcp_method", string(method)).Interface("id", id).
		Dur("elapsed", l.elapsed(id)).Msg("[mcp] error")
}

// elapsed returns and clears the time since the matching onBefore. It returns
// 0 when there was no paired onBefore (e.g. a parse error rejected before
// dispatch).
func (l *mcpLogger) elapsed(id any) time.Duration {
	if v, ok := l.starts.LoadAndDelete(id); ok {
		return time.Since(v.(time.Time))
	}
	return 0
}

// maxBodyLog caps how many bytes of a request body are written to the trace
// log; the full body is always restored for the handler.
const maxBodyLog = 4096

// logHTTP wraps an MCP transport handler to log the HTTP request and response,
// tagged with the transport ("streamable-http" or "sse") that served it.
// Verbosity is split like the rest of the logging:
//
//   - info:  one line per response — transport, method, path, client, status,
//     bytes and elapsed time.
//   - debug: adds a line per request — client IP, user agent, content length.
//   - trace: adds the request headers (credentials redacted) and body.
func logHTTP(transport string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		log.Debug().Str("transport", transport).Str("http_method", r.Method).
			Str("path", r.URL.Path).Str("remote", clientIP(r)).
			Str("user_agent", r.UserAgent()).Int64("req_bytes", r.ContentLength).
			Msg("[mcp] http request")

		if e := log.Trace(); e.Enabled() {
			e.Str("transport", transport).Str("http_method", r.Method).
				Str("path", r.URL.Path).Str("remote", clientIP(r)).
				Interface("headers", redactHTTPHeaders(r.Header)).
				Str("body", previewBody(r)).Msg("[mcp] http request detail")
		}

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		log.Info().Str("transport", transport).Str("http_method", r.Method).
			Str("path", r.URL.Path).Str("remote", clientIP(r)).
			Int("status", rec.status).Int("resp_bytes", rec.bytes).
			Dur("elapsed", time.Since(start)).Msg("[mcp] http response")
	})
}

// statusRecorder captures the status code and response size while preserving
// the streaming behaviour the MCP transports rely on (Flush for SSE, and
// Unwrap so http.ResponseController reaches the underlying writer).
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
	wrote  bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wrote {
		r.status = code
		r.wrote = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	r.wrote = true
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// clientIP returns the originating client address, honouring a reverse proxy's
// X-Forwarded-For / X-Real-IP before falling back to the direct peer.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		first, _, _ := strings.Cut(xff, ",")
		return strings.TrimSpace(first)
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return xr
	}
	return r.RemoteAddr
}

// previewBody reads the request body, restores it in full for the handler, and
// returns a size-capped copy for logging. Returns "" when there is no body.
func previewBody(r *http.Request) string {
	if r.Body == nil {
		return ""
	}
	data, err := io.ReadAll(r.Body)
	_ = r.Body.Close()
	if err != nil {
		return ""
	}
	r.Body = io.NopCloser(bytes.NewReader(data))
	if len(data) > maxBodyLog {
		return string(data[:maxBodyLog]) + "...(truncated)"
	}
	return string(data)
}

// redactHTTPHeaders copies h with credential-bearing headers masked so trace
// logging never leaks client secrets.
func redactHTTPHeaders(h http.Header) http.Header {
	out := h.Clone()
	if out == nil {
		return nil
	}
	for _, k := range []string{"Authorization", "Cookie", "X-Api-Key"} {
		if _, ok := out[k]; ok {
			out.Set(k, "REDACTED")
		}
	}
	return out
}

// contentSize reports the size of a response as the model would receive it.
// For tool results it sums the text content lengths directly (cheap even for
// multi-megabyte logs); other result types are small and measured by their
// JSON encoding.
func contentSize(result any) int {
	if r, ok := result.(*mcp.CallToolResult); ok {
		n := 0
		for _, c := range r.Content {
			if tc, ok := mcp.AsTextContent(c); ok {
				n += len(tc.Text)
			}
		}
		if r.StructuredContent != nil {
			if b, err := json.Marshal(r.StructuredContent); err == nil {
				n += len(b)
			}
		}
		return n
	}
	if b, err := json.Marshal(result); err == nil {
		return len(b)
	}
	return 0
}
