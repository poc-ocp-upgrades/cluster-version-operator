package resourcebuilder

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/openshift/cluster-version-operator/lib"
	"github.com/openshift/cluster-version-operator/lib/resourceapply"
	"github.com/openshift/cluster-version-operator/lib/resourceread"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	appsclientv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/rest"
)

type deploymentBuilder struct {
	client		*appsclientv1.AppsV1Client
	raw		[]byte
	modifier	MetaV1ObjectModifierFunc
}

func newDeploymentBuilder(config *rest.Config, m lib.Manifest) Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &deploymentBuilder{client: appsclientv1.NewForConfigOrDie(withProtobuf(config)), raw: m.Raw}
}
func (b *deploymentBuilder) WithMode(m Mode) Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return b
}
func (b *deploymentBuilder) WithModifier(f MetaV1ObjectModifierFunc) Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	b.modifier = f
	return b
}
func (b *deploymentBuilder) Do(ctx context.Context) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	deployment := resourceread.ReadDeploymentV1OrDie(b.raw)
	if b.modifier != nil {
		b.modifier(deployment)
	}
	actual, updated, err := resourceapply.ApplyDeployment(b.client, deployment)
	if err != nil {
		return err
	}
	if updated && actual.Generation > 1 {
		return waitForDeploymentCompletion(ctx, b.client, deployment)
	}
	return nil
}
func waitForDeploymentCompletion(ctx context.Context, client appsclientv1.DeploymentsGetter, deployment *appsv1.Deployment) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return wait.PollImmediateUntil(defaultObjectPollInterval, func() (bool, error) {
		d, err := client.Deployments(deployment.Namespace).Get(deployment.Name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return false, err
		}
		if err != nil {
			glog.Errorf("error getting Deployment %s during rollout: %v", deployment.Name, err)
			return false, nil
		}
		if d.DeletionTimestamp != nil {
			return false, fmt.Errorf("Deployment %s is being deleted", deployment.Name)
		}
		if d.Generation <= d.Status.ObservedGeneration && d.Status.UpdatedReplicas == d.Status.Replicas && d.Status.UnavailableReplicas == 0 {
			return true, nil
		}
		glog.V(4).Infof("Deployment %s is not ready. status: (replicas: %d, updated: %d, ready: %d, unavailable: %d)", d.Name, d.Status.Replicas, d.Status.UpdatedReplicas, d.Status.ReadyReplicas, d.Status.UnavailableReplicas)
		return false, nil
	}, ctx.Done())
}

type daemonsetBuilder struct {
	client		*appsclientv1.AppsV1Client
	raw		[]byte
	modifier	MetaV1ObjectModifierFunc
}

func newDaemonsetBuilder(config *rest.Config, m lib.Manifest) Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &daemonsetBuilder{client: appsclientv1.NewForConfigOrDie(withProtobuf(config)), raw: m.Raw}
}
func (b *daemonsetBuilder) WithMode(m Mode) Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return b
}
func (b *daemonsetBuilder) WithModifier(f MetaV1ObjectModifierFunc) Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	b.modifier = f
	return b
}
func (b *daemonsetBuilder) Do(ctx context.Context) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	daemonset := resourceread.ReadDaemonSetV1OrDie(b.raw)
	if b.modifier != nil {
		b.modifier(daemonset)
	}
	actual, updated, err := resourceapply.ApplyDaemonSet(b.client, daemonset)
	if err != nil {
		return err
	}
	if updated && actual.Generation > 1 {
		return waitForDaemonsetRollout(ctx, b.client, daemonset)
	}
	return nil
}
func waitForDaemonsetRollout(ctx context.Context, client appsclientv1.DaemonSetsGetter, daemonset *appsv1.DaemonSet) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return wait.PollImmediateUntil(defaultObjectPollInterval, func() (bool, error) {
		d, err := client.DaemonSets(daemonset.Namespace).Get(daemonset.Name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return false, err
		}
		if err != nil {
			glog.Errorf("error getting Daemonset %s during rollout: %v", daemonset.Name, err)
			return false, nil
		}
		if d.DeletionTimestamp != nil {
			return false, fmt.Errorf("Daemonset %s is being deleted", daemonset.Name)
		}
		if d.Generation <= d.Status.ObservedGeneration && d.Status.UpdatedNumberScheduled == d.Status.DesiredNumberScheduled && d.Status.NumberUnavailable == 0 {
			return true, nil
		}
		glog.V(4).Infof("Daemonset %s is not ready. status: (desired: %d, updated: %d, ready: %d, unavailable: %d)", d.Name, d.Status.DesiredNumberScheduled, d.Status.UpdatedNumberScheduled, d.Status.NumberReady, d.Status.NumberAvailable)
		return false, nil
	}, ctx.Done())
}
