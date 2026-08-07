package netboxx

import (
	"context"
	"net/url"
)

// ListClusters lists virtualization clusters.
func (s *Service) ListClusters(ctx context.Context, search, site string, brief bool, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", search)
	setRef(q, "site", site)
	setBrief(q, brief)
	return list(ctx, s, "virtualization/clusters", page, q)
}

// VMFilter narrows ListVirtualMachines.
type VMFilter struct {
	Search  string
	Cluster string
	Site    string
	Status  string
	Tenant  string
	Tag     string
	Brief   bool
}

// ListVirtualMachines lists VMs matching the filter.
func (s *Service) ListVirtualMachines(ctx context.Context, f VMFilter, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", f.Search)
	setRef(q, "cluster", f.Cluster)
	setRef(q, "site", f.Site)
	setStr(q, "status", f.Status)
	setRef(q, "tenant", f.Tenant)
	setStr(q, "tag", f.Tag)
	setBrief(q, f.Brief)
	return list(ctx, s, "virtualization/virtual-machines", page, q)
}

// GetVirtualMachine fetches one VM by numeric id or exact name.
func (s *Service) GetVirtualMachine(ctx context.Context, ref string) (Object, error) {
	return getOne(ctx, s, "virtualization/virtual-machines", ref, "name")
}

// ListVMInterfaces lists the interfaces of a VM (id or exact name).
func (s *Service) ListVMInterfaces(ctx context.Context, vm string, brief bool, page int) (*Page, error) {
	q := url.Values{}
	setRef(q, "virtual_machine", vm)
	setBrief(q, brief)
	return list(ctx, s, "virtualization/interfaces", page, q)
}
