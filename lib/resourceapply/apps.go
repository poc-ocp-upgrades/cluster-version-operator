package resourceapply

import (
	"github.com/openshift/cluster-version-operator/lib/resourcemerge"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsclientv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	appslisterv1 "k8s.io/client-go/listers/apps/v1"
	"k8s.io/utils/pointer"
)

func ApplyDeployment(client appsclientv1.DeploymentsGetter, required *appsv1.Deployment) (*appsv1.Deployment, bool, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	existing, err := client.Deployments(required.Namespace).Get(required.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		actual, err := client.Deployments(required.Namespace).Create(required)
		return actual, true, err
	}
	if err != nil {
		return nil, false, err
	}
	if IsCreateOnly(required) {
		return nil, false, nil
	}
	modified := pointer.BoolPtr(false)
	resourcemerge.EnsureDeployment(modified, existing, *required)
	if !*modified {
		return existing, false, nil
	}
	actual, err := client.Deployments(required.Namespace).Update(existing)
	return actual, true, err
}
func ApplyDeploymentFromCache(lister appslisterv1.DeploymentLister, client appsclientv1.DeploymentsGetter, required *appsv1.Deployment) (*appsv1.Deployment, bool, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	existing, err := lister.Deployments(required.Namespace).Get(required.Name)
	if apierrors.IsNotFound(err) {
		actual, err := client.Deployments(required.Namespace).Create(required)
		return actual, true, err
	}
	if err != nil {
		return nil, false, err
	}
	if IsCreateOnly(required) {
		return nil, false, nil
	}
	existing = existing.DeepCopy()
	modified := pointer.BoolPtr(false)
	resourcemerge.EnsureDeployment(modified, existing, *required)
	if !*modified {
		return existing, false, nil
	}
	actual, err := client.Deployments(required.Namespace).Update(existing)
	return actual, true, err
}
func ApplyDaemonSet(client appsclientv1.DaemonSetsGetter, required *appsv1.DaemonSet) (*appsv1.DaemonSet, bool, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	existing, err := client.DaemonSets(required.Namespace).Get(required.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		actual, err := client.DaemonSets(required.Namespace).Create(required)
		return actual, true, err
	}
	if err != nil {
		return nil, false, err
	}
	if IsCreateOnly(required) {
		return nil, false, nil
	}
	modified := pointer.BoolPtr(false)
	resourcemerge.EnsureDaemonSet(modified, existing, *required)
	if !*modified {
		return existing, false, nil
	}
	actual, err := client.DaemonSets(required.Namespace).Update(existing)
	return actual, true, err
}
func ApplyDaemonSetFromCache(lister appslisterv1.DaemonSetLister, client appsclientv1.DaemonSetsGetter, required *appsv1.DaemonSet) (*appsv1.DaemonSet, bool, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	existing, err := lister.DaemonSets(required.Namespace).Get(required.Name)
	if apierrors.IsNotFound(err) {
		actual, err := client.DaemonSets(required.Namespace).Create(required)
		return actual, true, err
	}
	if err != nil {
		return nil, false, err
	}
	if IsCreateOnly(required) {
		return nil, false, nil
	}
	existing = existing.DeepCopy()
	modified := pointer.BoolPtr(false)
	resourcemerge.EnsureDaemonSet(modified, existing, *required)
	if !*modified {
		return existing, false, nil
	}
	actual, err := client.DaemonSets(required.Namespace).Update(existing)
	return actual, true, err
}
