package resourcebuilder

import (
	"context"
	"github.com/openshift/cluster-version-operator/lib"
	"github.com/openshift/cluster-version-operator/lib/resourceapply"
	"github.com/openshift/cluster-version-operator/lib/resourceread"
	rbacclientv1 "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/rest"
)

type clusterRoleBuilder struct {
	client		*rbacclientv1.RbacV1Client
	raw		[]byte
	modifier	MetaV1ObjectModifierFunc
}

func newClusterRoleBuilder(config *rest.Config, m lib.Manifest) Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &clusterRoleBuilder{client: rbacclientv1.NewForConfigOrDie(withProtobuf(config)), raw: m.Raw}
}
func (b *clusterRoleBuilder) WithMode(m Mode) Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return b
}
func (b *clusterRoleBuilder) WithModifier(f MetaV1ObjectModifierFunc) Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	b.modifier = f
	return b
}
func (b *clusterRoleBuilder) Do(_ context.Context) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	clusterRole := resourceread.ReadClusterRoleV1OrDie(b.raw)
	if b.modifier != nil {
		b.modifier(clusterRole)
	}
	_, _, err := resourceapply.ApplyClusterRole(b.client, clusterRole)
	return err
}

type clusterRoleBindingBuilder struct {
	client		*rbacclientv1.RbacV1Client
	raw		[]byte
	modifier	MetaV1ObjectModifierFunc
}

func newClusterRoleBindingBuilder(config *rest.Config, m lib.Manifest) Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &clusterRoleBindingBuilder{client: rbacclientv1.NewForConfigOrDie(withProtobuf(config)), raw: m.Raw}
}
func (b *clusterRoleBindingBuilder) WithMode(m Mode) Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return b
}
func (b *clusterRoleBindingBuilder) WithModifier(f MetaV1ObjectModifierFunc) Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	b.modifier = f
	return b
}
func (b *clusterRoleBindingBuilder) Do(_ context.Context) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	clusterRoleBinding := resourceread.ReadClusterRoleBindingV1OrDie(b.raw)
	if b.modifier != nil {
		b.modifier(clusterRoleBinding)
	}
	_, _, err := resourceapply.ApplyClusterRoleBinding(b.client, clusterRoleBinding)
	return err
}

type roleBuilder struct {
	client		*rbacclientv1.RbacV1Client
	raw		[]byte
	modifier	MetaV1ObjectModifierFunc
}

func newRoleBuilder(config *rest.Config, m lib.Manifest) Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &roleBuilder{client: rbacclientv1.NewForConfigOrDie(withProtobuf(config)), raw: m.Raw}
}
func (b *roleBuilder) WithMode(m Mode) Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return b
}
func (b *roleBuilder) WithModifier(f MetaV1ObjectModifierFunc) Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	b.modifier = f
	return b
}
func (b *roleBuilder) Do(_ context.Context) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	role := resourceread.ReadRoleV1OrDie(b.raw)
	if b.modifier != nil {
		b.modifier(role)
	}
	_, _, err := resourceapply.ApplyRole(b.client, role)
	return err
}

type roleBindingBuilder struct {
	client		*rbacclientv1.RbacV1Client
	raw		[]byte
	modifier	MetaV1ObjectModifierFunc
}

func newRoleBindingBuilder(config *rest.Config, m lib.Manifest) Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &roleBindingBuilder{client: rbacclientv1.NewForConfigOrDie(withProtobuf(config)), raw: m.Raw}
}
func (b *roleBindingBuilder) WithMode(m Mode) Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return b
}
func (b *roleBindingBuilder) WithModifier(f MetaV1ObjectModifierFunc) Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	b.modifier = f
	return b
}
func (b *roleBindingBuilder) Do(_ context.Context) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	roleBinding := resourceread.ReadRoleBindingV1OrDie(b.raw)
	if b.modifier != nil {
		b.modifier(roleBinding)
	}
	_, _, err := resourceapply.ApplyRoleBinding(b.client, roleBinding)
	return err
}
