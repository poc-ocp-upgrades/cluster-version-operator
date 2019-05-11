package resourcebuilder

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/openshift/cluster-version-operator/lib"
	"github.com/openshift/cluster-version-operator/lib/resourceapply"
	"github.com/openshift/cluster-version-operator/lib/resourceread"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	batchclientv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	"k8s.io/client-go/rest"
)

type jobBuilder struct {
	client		*batchclientv1.BatchV1Client
	raw			[]byte
	modifier	MetaV1ObjectModifierFunc
}

func newJobBuilder(config *rest.Config, m lib.Manifest) Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &jobBuilder{client: batchclientv1.NewForConfigOrDie(withProtobuf(config)), raw: m.Raw}
}
func (b *jobBuilder) WithMode(m Mode) Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return b
}
func (b *jobBuilder) WithModifier(f MetaV1ObjectModifierFunc) Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	b.modifier = f
	return b
}
func (b *jobBuilder) Do(ctx context.Context) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	job := resourceread.ReadJobV1OrDie(b.raw)
	if b.modifier != nil {
		b.modifier(job)
	}
	_, updated, err := resourceapply.ApplyJob(b.client, job)
	if err != nil {
		return err
	}
	if updated {
		return WaitForJobCompletion(ctx, b.client, job)
	}
	return nil
}
func WaitForJobCompletion(ctx context.Context, client batchclientv1.JobsGetter, job *batchv1.Job) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return wait.PollImmediateUntil(defaultObjectPollInterval, func() (bool, error) {
		j, err := client.Jobs(job.Namespace).Get(job.Name, metav1.GetOptions{})
		if err != nil {
			glog.Errorf("error getting Job %s: %v", job.Name, err)
			return false, nil
		}
		if j.Status.Succeeded > 0 {
			return true, nil
		}
		if j.Status.Active == 0 && j.Status.Failed > 0 {
			reason := "DeadlineExceeded"
			message := "Job was active longer than specified deadline"
			if len(j.Status.Conditions) > 0 {
				reason, message = j.Status.Conditions[0].Reason, j.Status.Conditions[0].Message
			}
			return false, fmt.Errorf("deadline exceeded, reason: %q, message: %q", reason, message)
		}
		return false, nil
	}, ctx.Done())
}
