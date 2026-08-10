package netboxx

import (
	"context"
	"errors"
	"net/url"
)

// changelogPaths are the changelog list endpoints, newest NetBox first. The
// ObjectChange model moved from the extras app to core in NetBox 4.0, so the
// modern path is tried first and the legacy one only after a 404.
var changelogPaths = []string{"core/object-changes", "extras/object-changes"}

// ChangelogFilter narrows ListChangelog. User accepts a numeric id or a
// username; ObjectType is a NetBox app.model label such as "dcim.device".
// After and Before bound the change time and take anything NetBox parses as a
// datetime, e.g. "2026-08-01" or "2026-08-01T12:30:00Z".
type ChangelogFilter struct {
	Search     string
	User       string
	Action     string // created | updated | deleted
	ObjectType string
	ObjectID   string
	RequestID  string // groups every change made by one request
	After      string
	Before     string
	Brief      bool
}

// ListChangelog lists changelog records — NetBox's audit trail. Each record
// carries the acting user, the action, a snapshot of the object before and
// after (prechange_data/postchange_data) and the request id that groups
// changes made together.
func (s *Service) ListChangelog(ctx context.Context, f ChangelogFilter, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", f.Search)
	setRef(q, "user", f.User)
	setStr(q, "action", f.Action)
	setRef(q, "changed_object_type", f.ObjectType)
	setStr(q, "changed_object_id", f.ObjectID)
	setStr(q, "request_id", f.RequestID)
	setStr(q, "time_after", f.After)
	setStr(q, "time_before", f.Before)
	setBrief(q, f.Brief)

	var err error
	for _, path := range changelogPaths {
		var p *Page
		if p, err = list(ctx, s, path, page, q); err == nil {
			return p, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
	}
	return nil, err
}
