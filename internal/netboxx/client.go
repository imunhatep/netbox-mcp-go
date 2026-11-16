// Package netboxx is a lightweight read-only client for the NetBox REST API.
// Objects are passed through as returned by NetBox — including custom_fields,
// tags and comments — so no data is lost to hand-maintained DTOs. Only the
// hyperlink noise (url/display_url keys) is stripped to keep output compact.
package netboxx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/imunhatep/netbox-mcp-go/internal/config"
)

// Object is a NetBox API object, passed through as-is.
type Object = map[string]any

// Service wraps an HTTP client bound to one NetBox instance.
type Service struct {
	httpClient *http.Client
	baseURL    string // e.g. https://netbox.example.com (no /api suffix)
	token      string
	pageSize   int
}

// NewService constructs a NetBox API client from configuration.
func NewService(cfg *config.Config) (*Service, error) {
	return &Service{
		httpClient: newLoggingHTTPClient(cfg.RequestTimeout),
		baseURL:    cfg.NetBoxURL,
		token:      cfg.NetBoxToken,
		pageSize:   cfg.PageSize,
	}, nil
}

// get performs GET {base}/api/{path}?{query} and decodes the JSON body into out.
func (s *Service) get(ctx context.Context, path string, query url.Values, out any) error {
	u := s.baseURL + "/api/" + strings.TrimLeft(path, "/")
	if !strings.HasSuffix(u, "/") {
		u += "/"
	}
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	token, err := s.resolveToken(ctx)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", authHeader(token))
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apiError(resp)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// listEnvelope is the raw NetBox paginated response.
type listEnvelope struct {
	Count   int      `json:"count"`
	Next    *string  `json:"next"`
	Results []Object `json:"results"`
}

// Page is the shape list tools return to the model: total count plus the
// requested 1-based page of results.
type Page struct {
	Count    int      `json:"count"`
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
	HasNext  bool     `json:"has_next"`
	Results  []Object `json:"results"`
}

// list fetches one page of a NetBox list endpoint. page is 1-based; extra
// filter params are merged into the query.
func list(ctx context.Context, s *Service, path string, page int, query url.Values) (*Page, error) {
	if page < 1 {
		page = 1
	}
	if query == nil {
		query = url.Values{}
	}
	query.Set("limit", strconv.Itoa(s.pageSize))
	query.Set("offset", strconv.Itoa((page-1)*s.pageSize))

	var env listEnvelope
	if err := s.get(ctx, path, query, &env); err != nil {
		return nil, err
	}
	for _, obj := range env.Results {
		prune(obj)
	}
	return &Page{
		Count:    env.Count,
		Page:     page,
		PageSize: s.pageSize,
		HasNext:  env.Next != nil,
		Results:  env.Results,
	}, nil
}

// getOne fetches a single object either by numeric id (GET {path}/{id}/) or by
// a unique lookup filter (?{lookupParam}={ref}); the lookup must match exactly
// one object.
func getOne(ctx context.Context, s *Service, path, ref, lookupParam string) (Object, error) {
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		var obj Object
		if err := s.get(ctx, fmt.Sprintf("%s/%d", path, id), nil, &obj); err != nil {
			return nil, err
		}
		prune(obj)
		return obj, nil
	}

	q := url.Values{}
	q.Set(lookupParam, ref)
	q.Set("limit", "2")
	var env listEnvelope
	if err := s.get(ctx, path, q, &env); err != nil {
		return nil, err
	}
	switch len(env.Results) {
	case 0:
		return nil, fmt.Errorf("%w: no %s with %s=%q", ErrNotFound, path, lookupParam, ref)
	case 1:
		prune(env.Results[0])
		return env.Results[0], nil
	default:
		return nil, fmt.Errorf("%w: %s=%q matches %d objects in %s, use a numeric id",
			ErrValidation, lookupParam, ref, env.Count, path)
	}
}

// prune recursively strips hyperlink keys (url, display_url) that NetBox adds
// to every object and nested reference. They carry no information a model can
// use and inflate output; all data fields are kept verbatim.
func prune(v any) {
	switch t := v.(type) {
	case map[string]any:
		delete(t, "url")
		delete(t, "display_url")
		for _, vv := range t {
			prune(vv)
		}
	case []any:
		for _, vv := range t {
			prune(vv)
		}
	}
}

// setRef adds a filter that accepts either a numeric id or a symbolic
// reference: "site_id=5" when ref is numeric, otherwise "site=<slug>".
func setRef(q url.Values, param, ref string) {
	if ref == "" {
		return
	}
	if _, err := strconv.ParseInt(ref, 10, 64); err == nil {
		q.Set(param+"_id", ref)
	} else {
		q.Set(param, ref)
	}
}

// setStr adds a plain string filter when non-empty.
func setStr(q url.Values, param, val string) {
	if val != "" {
		q.Set(param, val)
	}
}

// setBrief switches a list request to NetBox's brief representation (id,
// name/display and little else) — useful for broad listings.
func setBrief(q url.Values, brief bool) {
	if brief {
		q.Set("brief", "true")
	}
}

// setMulti adds a repeated filter (id=1&id=2) for NetBox's multi-value
// filters, which OR the values together. Blank values are skipped.
func setMulti(q url.Values, param string, vals []string) {
	for _, v := range vals {
		if v = strings.TrimSpace(v); v != "" {
			q.Add(param, v)
		}
	}
}

// setFields restricts a list response to the named top-level fields via
// NetBox's ?fields= parameter (NetBox 4.0+), which takes precedence over
// brief. id is always included. NetBox matches top-level names only and
// silently ignores anything else, so a nested path such as "site.name" is
// rejected here rather than quietly returning nothing.
func setFields(q url.Values, fields []string) error {
	if len(fields) == 0 {
		return nil
	}
	out := []string{"id"}
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if strings.ContainsAny(f, ". ") {
			return fmt.Errorf("%w: field %q: only top-level field names are supported, e.g. site or primary_ip4", ErrValidation, f)
		}
		if f != "" && f != "id" {
			out = append(out, f)
		}
	}
	q.Set("fields", strings.Join(out, ","))
	q.Del("brief")
	return nil
}
