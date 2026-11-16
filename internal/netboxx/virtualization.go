package netboxx

import (
	"context"
	"net/url"
)

// ClusterFilter narrows ListClusters. Reference fields accept a numeric id or
// slug.
type ClusterFilter struct {
	Search string
	Site   string
	Type   string // cluster type id or slug
	Group  string // cluster group id or slug
	Tenant string
	Status string
	Tag    string
	Fields []string // top-level fields to return; overrides Brief
	Brief  bool
}

// ListClusters lists virtualization clusters matching the filter.
func (s *Service) ListClusters(ctx context.Context, f ClusterFilter, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", f.Search)
	setRef(q, "site", f.Site)
	setRef(q, "type", f.Type)
	setRef(q, "group", f.Group)
	setRef(q, "tenant", f.Tenant)
	setStr(q, "status", f.Status)
	setStr(q, "tag", f.Tag)
	setBrief(q, f.Brief)
	if err := setFields(q, f.Fields); err != nil {
		return nil, err
	}
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
	IDs     []string // exact numeric ids, OR-ed: one call instead of a get per VM
	Names   []string // exact names, OR-ed
	Fields  []string // top-level fields to return; overrides Brief
	Brief   bool
}

// ListVirtualMachines lists VMs matching the filter.
func (s *Service) ListVirtualMachines(ctx context.Context, f VMFilter, page int) (*Page, error) {
	q := url.Values{}
	setMulti(q, "id", f.IDs)
	setMulti(q, "name", f.Names)
	setStr(q, "q", f.Search)
	setRef(q, "cluster", f.Cluster)
	setRef(q, "site", f.Site)
	setStr(q, "status", f.Status)
	setRef(q, "tenant", f.Tenant)
	setStr(q, "tag", f.Tag)
	setBrief(q, f.Brief)
	if err := setFields(q, f.Fields); err != nil {
		return nil, err
	}
	return list(ctx, s, "virtualization/virtual-machines", page, q)
}

// GetVirtualMachine fetches one VM by numeric id or exact name.
func (s *Service) GetVirtualMachine(ctx context.Context, ref string) (Object, error) {
	return getOne(ctx, s, "virtualization/virtual-machines", ref, "name")
}

// VMInterfaceFilter narrows ListVMInterfaces. VM is required and accepts a
// numeric id or exact name.
type VMInterfaceFilter struct {
	VM    string
	Brief bool
}

// ListVMInterfaces lists the interfaces of one virtual machine.
func (s *Service) ListVMInterfaces(ctx context.Context, f VMInterfaceFilter, page int) (*Page, error) {
	q := url.Values{}
	setRef(q, "virtual_machine", f.VM)
	setBrief(q, f.Brief)
	return list(ctx, s, "virtualization/interfaces", page, q)
}
