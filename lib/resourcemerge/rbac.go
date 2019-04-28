package resourcemerge

import (
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func EnsureClusterRoleBinding(modified *bool, existing *rbacv1.ClusterRoleBinding, required rbacv1.ClusterRoleBinding) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	EnsureObjectMeta(modified, &existing.ObjectMeta, required.ObjectMeta)
	if !equality.Semantic.DeepEqual(existing.Subjects, required.Subjects) {
		*modified = true
		existing.Subjects = required.Subjects
	}
	if !equality.Semantic.DeepEqual(existing.RoleRef, required.RoleRef) {
		*modified = true
		existing.RoleRef = required.RoleRef
	}
}
func EnsureClusterRole(modified *bool, existing *rbacv1.ClusterRole, required rbacv1.ClusterRole) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	EnsureObjectMeta(modified, &existing.ObjectMeta, required.ObjectMeta)
	if !equality.Semantic.DeepEqual(existing.Rules, required.Rules) {
		*modified = true
		existing.Rules = required.Rules
	}
}
func EnsureRoleBinding(modified *bool, existing *rbacv1.RoleBinding, required rbacv1.RoleBinding) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	EnsureObjectMeta(modified, &existing.ObjectMeta, required.ObjectMeta)
	if !equality.Semantic.DeepEqual(existing.Subjects, required.Subjects) {
		*modified = true
		existing.Subjects = required.Subjects
	}
	if !equality.Semantic.DeepEqual(existing.RoleRef, required.RoleRef) {
		*modified = true
		existing.RoleRef = required.RoleRef
	}
}
func EnsureRole(modified *bool, existing *rbacv1.Role, required rbacv1.Role) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	EnsureObjectMeta(modified, &existing.ObjectMeta, required.ObjectMeta)
	if !equality.Semantic.DeepEqual(existing.Rules, required.Rules) {
		*modified = true
		existing.Rules = required.Rules
	}
}
