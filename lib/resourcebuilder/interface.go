package resourcebuilder

import (
	"context"
	"fmt"
	"sync"
	"time"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"github.com/openshift/cluster-version-operator/lib"
)

var (
	Mapper = NewResourceMapper()
)

type ResourceMapper struct {
	l		*sync.Mutex
	gvkToNew	map[schema.GroupVersionKind]NewInteraceFunc
}

func (rm *ResourceMapper) AddToMap(irm *ResourceMapper) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	irm.l.Lock()
	defer irm.l.Unlock()
	for k, v := range rm.gvkToNew {
		irm.gvkToNew[k] = v
	}
}
func (rm *ResourceMapper) Exists(gvk schema.GroupVersionKind) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_, ok := rm.gvkToNew[gvk]
	return ok
}
func (rm *ResourceMapper) RegisterGVK(gvk schema.GroupVersionKind, f NewInteraceFunc) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	rm.gvkToNew[gvk] = f
}
func NewResourceMapper() *ResourceMapper {
	_logClusterCodePath()
	defer _logClusterCodePath()
	m := map[schema.GroupVersionKind]NewInteraceFunc{}
	return &ResourceMapper{l: &sync.Mutex{}, gvkToNew: m}
}

type MetaV1ObjectModifierFunc func(metav1.Object)
type NewInteraceFunc func(rest *rest.Config, m lib.Manifest) Interface
type Mode int

const (
	UpdatingMode	Mode	= iota
	ReconcilingMode
	InitializingMode
)

type Interface interface {
	WithModifier(MetaV1ObjectModifierFunc) Interface
	WithMode(Mode) Interface
	Do(context.Context) error
}

func New(mapper *ResourceMapper, rest *rest.Config, m lib.Manifest) (Interface, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	f, ok := mapper.gvkToNew[m.GVK]
	if !ok {
		return nil, fmt.Errorf("No mapping found for gvk: %v", m.GVK)
	}
	return f(rest, m), nil
}

const defaultObjectPollInterval = 3 * time.Second
