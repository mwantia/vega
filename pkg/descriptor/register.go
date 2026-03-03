package descriptor

import (
	"sync"

	"github.com/mwantia/vega/pkg/slot"
)

var Global DescriptorRegister = NewRegister()

type descriptorRegisterImpl struct {
	mu sync.RWMutex

	methods  map[slot.TypeTag]map[string]*MethodDescriptor
	members  map[slot.TypeTag]map[string]*MemberDescriptor
	stencils map[string]*StencilDescriptor
	statics  map[string]*MethodDescriptor
}

func NewRegister() DescriptorRegister {
	return &descriptorRegisterImpl{
		methods:  make(map[slot.TypeTag]map[string]*MethodDescriptor),
		members:  make(map[slot.TypeTag]map[string]*MemberDescriptor),
		stencils: make(map[string]*StencilDescriptor),
		statics:  make(map[string]*MethodDescriptor),
	}
}

func (dr *descriptorRegisterImpl) RegisterMethod(tag slot.TypeTag, desc *MethodDescriptor) error {
	dr.mu.Lock()
	defer dr.mu.Unlock()

	if dr.methods[tag] == nil {
		dr.methods[tag] = make(map[string]*MethodDescriptor)
	}
	dr.methods[tag][desc.Name] = desc
	return nil
}

func (dr *descriptorRegisterImpl) RegisterStatic(desc *MethodDescriptor) error {
	dr.mu.Lock()
	defer dr.mu.Unlock()

	dr.statics[desc.Name] = desc
	return nil
}

func (dr *descriptorRegisterImpl) RegisterMember(tag slot.TypeTag, desc *MemberDescriptor) error {
	dr.mu.Lock()
	defer dr.mu.Unlock()

	if dr.members[tag] == nil {
		dr.members[tag] = make(map[string]*MemberDescriptor)
	}
	dr.members[tag][desc.Name] = desc
	return nil
}

func (dr *descriptorRegisterImpl) RegisterStencil(desc *StencilDescriptor) error {
	dr.mu.Lock()
	defer dr.mu.Unlock()

	dr.stencils[desc.Name] = desc
	return nil
}

func (dr *descriptorRegisterImpl) LookupMethod(tag slot.TypeTag, name string) (*MethodDescriptor, bool) {
	dr.mu.RLock()
	defer dr.mu.RUnlock()

	if m, ok := dr.methods[tag]; ok {
		desc, ok := m[name]
		return desc, ok
	}
	return nil, false
}

func (dr *descriptorRegisterImpl) LookupStatic(name string) (*MethodDescriptor, bool) {
	dr.mu.RLock()
	defer dr.mu.RUnlock()

	desc, ok := dr.statics[name]
	return desc, ok
}

func (dr *descriptorRegisterImpl) LookupMember(tag slot.TypeTag, name string) (*MemberDescriptor, bool) {
	dr.mu.RLock()
	defer dr.mu.RUnlock()

	if m, ok := dr.members[tag]; ok {
		desc, ok := m[name]
		return desc, ok
	}
	return nil, false
}

func (dr *descriptorRegisterImpl) LookupStencil(name string) (*StencilDescriptor, bool) {
	dr.mu.RLock()
	defer dr.mu.RUnlock()

	desc, ok := dr.stencils[name]
	return desc, ok
}

var _ DescriptorRegister = (*descriptorRegisterImpl)(nil)
