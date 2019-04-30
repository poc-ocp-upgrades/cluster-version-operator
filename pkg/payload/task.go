package payload

import (
	"context"
	"fmt"
	"strings"
	"time"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"github.com/openshift/cluster-version-operator/lib"
)

var (
	metricPayloadErrors = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "cluster_operator_payload_errors", Help: "Report the number of errors encountered applying the payload."}, []string{"version"})
)

func init() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	prometheus.MustRegister(metricPayloadErrors)
}

type ResourceBuilder interface {
	Apply(context.Context, *lib.Manifest, State) error
}
type Task struct {
	Index		int
	Total		int
	Manifest	*lib.Manifest
	Requeued	int
	Backoff		wait.Backoff
}

func (st *Task) Copy() *Task {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &Task{Index: st.Index, Total: st.Total, Manifest: st.Manifest, Requeued: st.Requeued}
}
func (st *Task) String() string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	ns := st.Manifest.Object().GetNamespace()
	if len(ns) == 0 {
		return fmt.Sprintf("%s %q (%d of %d)", strings.ToLower(st.Manifest.GVK.Kind), st.Manifest.Object().GetName(), st.Index, st.Total)
	}
	return fmt.Sprintf("%s \"%s/%s\" (%d of %d)", strings.ToLower(st.Manifest.GVK.Kind), ns, st.Manifest.Object().GetName(), st.Index, st.Total)
}
func (st *Task) Run(ctx context.Context, version string, builder ResourceBuilder, state State) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	var lastErr error
	backoff := st.Backoff
	maxDuration := 15 * time.Second
	for {
		err := builder.Apply(ctx, st.Manifest, state)
		if err == nil {
			return nil
		}
		lastErr = err
		utilruntime.HandleError(errors.Wrapf(err, "error running apply for %s", st))
		metricPayloadErrors.WithLabelValues(version).Inc()
		d := time.Duration(float64(backoff.Duration) * backoff.Factor)
		if d > maxDuration {
			d = maxDuration
		}
		d = wait.Jitter(d, backoff.Jitter)
		select {
		case <-time.After(d):
			continue
		case <-ctx.Done():
			if uerr, ok := lastErr.(*UpdateError); ok {
				uerr.Task = st.Copy()
				return uerr
			}
			reason, cause := reasonForPayloadSyncError(lastErr)
			if len(cause) > 0 {
				cause = ": " + cause
			}
			return &UpdateError{Nested: lastErr, Reason: reason, Message: fmt.Sprintf("Could not update %s%s", st, cause), Task: st.Copy()}
		}
	}
}

type UpdateError struct {
	Nested	error
	Reason	string
	Message	string
	Name	string
	Task	*Task
}

func (e *UpdateError) Error() string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return e.Message
}
func (e *UpdateError) Cause() error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return e.Nested
}
func reasonForPayloadSyncError(err error) (string, string) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	err = errors.Cause(err)
	switch {
	case apierrors.IsNotFound(err), apierrors.IsAlreadyExists(err):
		return "UpdatePayloadResourceNotFound", "resource may have been deleted"
	case apierrors.IsConflict(err):
		return "UpdatePayloadResourceConflict", "someone else is updating this resource"
	case apierrors.IsTimeout(err), apierrors.IsServiceUnavailable(err), apierrors.IsUnexpectedServerError(err):
		return "UpdatePayloadClusterDown", "the server is down or not responding"
	case apierrors.IsInternalError(err):
		return "UpdatePayloadClusterError", "the server is reporting an internal error"
	case apierrors.IsInvalid(err):
		return "UpdatePayloadResourceInvalid", "the object is invalid, possibly due to local cluster configuration"
	case apierrors.IsUnauthorized(err):
		return "UpdatePayloadClusterUnauthorized", "could not authenticate to the server"
	case apierrors.IsForbidden(err):
		return "UpdatePayloadResourceForbidden", "the server has forbidden updates to this resource"
	case apierrors.IsServerTimeout(err), apierrors.IsTooManyRequests(err):
		return "UpdatePayloadClusterOverloaded", "the server is overloaded and is not accepting updates"
	case meta.IsNoMatchError(err):
		return "UpdatePayloadResourceTypeMissing", "the server does not recognize this resource, check extension API servers"
	default:
		return "UpdatePayloadFailed", ""
	}
}
func SummaryForReason(reason, name string) string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	switch reason {
	case "UpdatePayloadResourceNotFound", "UpdatePayloadResourceConflict":
		return "some resources could not be updated"
	case "UpdatePayloadClusterDown":
		return "the control plane is down or not responding"
	case "UpdatePayloadClusterError":
		return "the control plane is reporting an internal error"
	case "UpdatePayloadClusterOverloaded":
		return "the control plane is overloaded and is not accepting updates"
	case "UpdatePayloadClusterUnauthorized":
		return "could not authenticate to the server"
	case "UpdatePayloadRetrievalFailed":
		return "could not download the update"
	case "UpdatePayloadResourceForbidden":
		return "the server is rejecting updates"
	case "UpdatePayloadResourceTypeMissing":
		return "a required extension is not available to update"
	case "UpdatePayloadResourceInvalid":
		return "some cluster configuration is invalid"
	case "UpdatePayloadIntegrity":
		return "the contents of the update are invalid"
	case "ImageVerificationFailed":
		return "the image may not be safe to use"
	case "ClusterOperatorDegraded":
		if len(name) > 0 {
			return fmt.Sprintf("the cluster operator %s is degraded", name)
		}
		return "a cluster operator is degraded"
	case "ClusterOperatorNotAvailable":
		if len(name) > 0 {
			return fmt.Sprintf("the cluster operator %s has not yet successfully rolled out", name)
		}
		return "a cluster operator has not yet rolled out"
	case "ClusterOperatorsNotAvailable":
		return "some cluster operators have not yet rolled out"
	}
	if strings.HasPrefix(reason, "UpdatePayload") {
		return "the update could not be applied"
	}
	return "an unknown error has occurred"
}
