package netboxx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/imunhatep/netbox-mcp-go/internal/config"
)

func testService(t *testing.T, handler http.HandlerFunc) *Service {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	svc, err := NewService(&config.Config{
		NetBoxURL:      srv.URL,
		NetBoxToken:    "tok",
		RequestTimeout: 5 * time.Second,
		PageSize:       2,
	})
	require.NoError(t, err)
	return svc
}

func TestListPagination(t *testing.T) {
	var gotPath, gotQuery string
	var gotAuth string
	svc := testService(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count": 5, "next": "http://x/api/dcim/sites/?limit=2&offset=4",
			"results": [
				{"id":3,"name":"c","slug":"c","url":"http://x/api/dcim/sites/3/",
				 "custom_fields":{"owner":"noc"},"tags":[{"id":1,"name":"prod","url":"http://x/api/extras/tags/1/"}]},
				{"id":4,"name":"d","slug":"d"}]}`))
	})

	page, err := svc.ListSites(context.Background(), SiteFilter{Search: "dc"}, 2)
	require.NoError(t, err)

	assert.Equal(t, "/api/dcim/sites/", gotPath)
	assert.Equal(t, "Token tok", gotAuth)
	q, err := url.ParseQuery(gotQuery)
	require.NoError(t, err)
	assert.Equal(t, "2", q.Get("limit"))
	assert.Equal(t, "2", q.Get("offset"))
	assert.Equal(t, "dc", q.Get("q"))
	assert.False(t, q.Has("brief"))

	assert.Equal(t, 5, page.Count)
	assert.Equal(t, 2, page.Page)
	assert.True(t, page.HasNext)
	require.Len(t, page.Results, 2)

	site := page.Results[0]
	assert.Equal(t, "c", site["name"])
	// Full passthrough: custom_fields and tags survive...
	assert.Equal(t, map[string]any{"owner": "noc"}, site["custom_fields"])
	tags, ok := site["tags"].([]any)
	require.True(t, ok)
	require.Len(t, tags, 1)
	// ...but hyperlink noise is pruned, including inside nested objects.
	assert.NotContains(t, site, "url")
	assert.NotContains(t, tags[0], "url")
}

func TestListBrief(t *testing.T) {
	svc := testService(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "true", r.URL.Query().Get("brief"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":0,"next":null,"results":[]}`))
	})

	_, err := svc.ListSites(context.Background(), SiteFilter{Brief: true}, 1)
	require.NoError(t, err)
}

// TestListSitesFilters pins the id-or-slug contract on the organization
// filters: numeric refs become <param>_id, symbolic ones stay <param>.
func TestListSitesFilters(t *testing.T) {
	var got url.Values
	svc := testService(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":0,"next":null,"results":[]}`))
	})

	_, err := svc.ListSites(context.Background(), SiteFilter{
		Region: "western-europe",
		Group:  "3",
		Tenant: "acme-corp",
		Status: "active",
	}, 1)
	require.NoError(t, err)

	assert.Equal(t, "western-europe", got.Get("region"))
	assert.Equal(t, "3", got.Get("group_id"))
	assert.False(t, got.Has("group"))
	assert.Equal(t, "acme-corp", got.Get("tenant"))
	assert.Equal(t, "active", got.Get("status"))
}

// TestOrganizationEndpoints asserts each new organization list/get hits the
// right NetBox path with the right hierarchy filters.
func TestOrganizationEndpoints(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	svc := testService(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":1,"next":null,"results":[{"id":1,"name":"x","slug":"x",` +
			`"url":"http://x/api/ignored/1/","custom_fields":{"cmdb":"1"}}]}`))
	})

	cases := []struct {
		name      string
		call      func() error
		wantPath  string
		wantQuery map[string]string
	}{
		{
			name:      "list regions by parent slug",
			call:      func() error { _, err := svc.ListRegions(context.Background(), "eu", "emea", false, 1); return err },
			wantPath:  "/api/dcim/regions/",
			wantQuery: map[string]string{"q": "eu", "parent": "emea"},
		},
		{
			name:      "list site groups by numeric parent",
			call:      func() error { _, err := svc.ListSiteGroups(context.Background(), "", "7", false, 1); return err },
			wantPath:  "/api/dcim/site-groups/",
			wantQuery: map[string]string{"parent_id": "7"},
		},
		{
			name: "list locations by site and parent",
			call: func() error {
				_, err := svc.ListLocations(context.Background(), "", "ams-dc1", "floor2", true, 1)
				return err
			},
			wantPath:  "/api/dcim/locations/",
			wantQuery: map[string]string{"site": "ams-dc1", "parent": "floor2", "brief": "true"},
		},
		{
			name:      "list racks by location",
			call:      func() error { _, err := svc.ListRacks(context.Background(), "", "", "cage-4", false, 1); return err },
			wantPath:  "/api/dcim/racks/",
			wantQuery: map[string]string{"location": "cage-4"},
		},
		{
			name:      "list tenant groups",
			call:      func() error { _, err := svc.ListTenantGroups(context.Background(), "biz", "", false, 1); return err },
			wantPath:  "/api/tenancy/tenant-groups/",
			wantQuery: map[string]string{"q": "biz"},
		},
		{
			name:      "list tenants by group",
			call:      func() error { _, err := svc.ListTenants(context.Background(), "", "internal", false, 1); return err },
			wantPath:  "/api/tenancy/tenants/",
			wantQuery: map[string]string{"group": "internal"},
		},
		{
			name:      "list contacts by group",
			call:      func() error { _, err := svc.ListContacts(context.Background(), "noc", "4", false, 1); return err },
			wantPath:  "/api/tenancy/contacts/",
			wantQuery: map[string]string{"q": "noc", "group_id": "4"},
		},
		{
			name:      "get region by slug",
			call:      func() error { _, err := svc.GetRegion(context.Background(), "western-europe"); return err },
			wantPath:  "/api/dcim/regions/",
			wantQuery: map[string]string{"slug": "western-europe"},
		},
		{
			name:      "get site group by id",
			call:      func() error { _, err := svc.GetSiteGroup(context.Background(), "2"); return err },
			wantPath:  "/api/dcim/site-groups/2/",
			wantQuery: map[string]string{},
		},
		{
			name:      "get location by slug",
			call:      func() error { _, err := svc.GetLocation(context.Background(), "floor2"); return err },
			wantPath:  "/api/dcim/locations/",
			wantQuery: map[string]string{"slug": "floor2"},
		},
		{
			name:      "get tenant by slug",
			call:      func() error { _, err := svc.GetTenant(context.Background(), "acme-corp"); return err },
			wantPath:  "/api/tenancy/tenants/",
			wantQuery: map[string]string{"slug": "acme-corp"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, tc.call())
			assert.Equal(t, tc.wantPath, gotPath)
			for k, v := range tc.wantQuery {
				assert.Equal(t, v, gotQuery.Get(k), "query param %q", k)
			}
		})
	}
}

// TestGetRegionPassthrough checks the new endpoints go through prune() and keep
// their data fields, like every other object.
func TestGetRegionPassthrough(t *testing.T) {
	svc := testService(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":3,"name":"Western Europe","slug":"western-europe",` +
			`"parent":{"id":1,"name":"EMEA","url":"http://x/api/dcim/regions/1/"},` +
			`"site_count":9,"url":"http://x/api/dcim/regions/3/","custom_fields":{"iso":"EU"}}`))
	})

	region, err := svc.GetRegion(context.Background(), "3")
	require.NoError(t, err)
	assert.Equal(t, "Western Europe", region["name"])
	assert.Equal(t, float64(9), region["site_count"])
	assert.Equal(t, map[string]any{"iso": "EU"}, region["custom_fields"])
	assert.NotContains(t, region, "url")
	parent, ok := region["parent"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "EMEA", parent["name"])
	assert.NotContains(t, parent, "url")
}

func TestGetOneByID(t *testing.T) {
	svc := testService(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/dcim/devices/42/", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":42,"name":"sw-core-01","comments":"replaced PSU 2024"}`))
	})

	dev, err := svc.GetDevice(context.Background(), "42")
	require.NoError(t, err)
	assert.Equal(t, float64(42), dev["id"])
	assert.Equal(t, "sw-core-01", dev["name"])
	assert.Equal(t, "replaced PSU 2024", dev["comments"])
}

func TestGetOneByName(t *testing.T) {
	svc := testService(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/dcim/devices/", r.URL.Path)
		assert.Equal(t, "sw-core-01", r.URL.Query().Get("name"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":1,"next":null,"results":[{"id":42,"name":"sw-core-01"}]}`))
	})

	dev, err := svc.GetDevice(context.Background(), "sw-core-01")
	require.NoError(t, err)
	assert.Equal(t, float64(42), dev["id"])
}

func TestGetOneNotFound(t *testing.T) {
	svc := testService(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":0,"next":null,"results":[]}`))
	})

	_, err := svc.GetDevice(context.Background(), "nope")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestGetOneAmbiguous(t *testing.T) {
	svc := testService(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":2,"next":null,"results":[{"id":1,"name":"a"},{"id":2,"name":"a"}]}`))
	})

	_, err := svc.GetDevice(context.Background(), "a")
	require.ErrorIs(t, err, ErrValidation)
}

func TestAPIErrorMapping(t *testing.T) {
	cases := []struct {
		status int
		want   error
	}{
		{http.StatusNotFound, ErrNotFound},
		{http.StatusUnauthorized, ErrUnauthorized},
		{http.StatusForbidden, ErrForbidden},
	}
	for _, tc := range cases {
		svc := testService(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(`{"detail":"boom"}`))
		})
		_, err := svc.GetSite(context.Background(), "5")
		require.ErrorIs(t, err, tc.want, "status %d", tc.status)
		assert.Contains(t, err.Error(), "boom")
	}
}

func TestSearch(t *testing.T) {
	svc := testService(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "core", r.URL.Query().Get("q"))
		assert.Equal(t, "true", r.URL.Query().Get("brief"))
		assert.Equal(t, "5", r.URL.Query().Get("limit"))
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/dcim/devices/" {
			_, _ = w.Write([]byte(`{"count":3,"next":null,"results":[{"id":42,"name":"sw-core-01","url":"x"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"count":0,"next":null,"results":[]}`))
	})

	groups, err := svc.Search(context.Background(), "core", 5)
	require.NoError(t, err)

	// Only the object type with matches is returned; empty groups are dropped.
	require.Len(t, groups, 1)
	assert.Equal(t, "devices", groups[0].ObjectType)
	assert.Equal(t, 3, groups[0].Count)
	require.Len(t, groups[0].Results, 1)
	assert.Equal(t, "sw-core-01", groups[0].Results[0]["name"])
	assert.NotContains(t, groups[0].Results[0], "url")
}

func TestSearchPartialFailure(t *testing.T) {
	svc := testService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/ipam/vrfs/" {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"detail":"no perms"}`))
			return
		}
		_, _ = w.Write([]byte(`{"count":0,"next":null,"results":[]}`))
	})

	_, err := svc.Search(context.Background(), "x", 5)
	require.ErrorIs(t, err, ErrForbidden)
	assert.Contains(t, err.Error(), "vrfs")
}

func TestSetRef(t *testing.T) {
	q := url.Values{}
	setRef(q, "site", "ams-dc1")
	setRef(q, "rack", "17")
	setRef(q, "role", "")

	assert.Equal(t, "ams-dc1", q.Get("site"))
	assert.Equal(t, "17", q.Get("rack_id"))
	assert.Empty(t, q.Get("rack"))
	assert.False(t, q.Has("role"))
	assert.False(t, q.Has("role_id"))
}
