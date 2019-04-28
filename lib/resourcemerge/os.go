package resourcemerge

import (
	"time"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	configv1 "github.com/openshift/api/config/v1"
)

func EnsureClusterOperatorStatus(modified *bool, existing *configv1.ClusterOperator, required configv1.ClusterOperator) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	EnsureObjectMeta(modified, &existing.ObjectMeta, required.ObjectMeta)
	ensureClusterOperatorStatus(modified, &existing.Status, required.Status)
}
func ensureClusterOperatorStatus(modified *bool, existing *configv1.ClusterOperatorStatus, required configv1.ClusterOperatorStatus) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if !equality.Semantic.DeepEqual(existing.Conditions, required.Conditions) {
		*modified = true
		existing.Conditions = required.Conditions
	}
	if !equality.Semantic.DeepEqual(existing.Versions, required.Versions) {
		*modified = true
		existing.Versions = required.Versions
	}
	if !equality.Semantic.DeepEqual(existing.Extension.Raw, required.Extension.Raw) {
		*modified = true
		existing.Extension.Raw = required.Extension.Raw
	}
	if !equality.Semantic.DeepEqual(existing.Extension.Object, required.Extension.Object) {
		*modified = true
		existing.Extension.Object = required.Extension.Object
	}
	if !equality.Semantic.DeepEqual(existing.RelatedObjects, required.RelatedObjects) {
		*modified = true
		existing.RelatedObjects = required.RelatedObjects
	}
}
func SetOperatorStatusCondition(conditions *[]configv1.ClusterOperatorStatusCondition, newCondition configv1.ClusterOperatorStatusCondition) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if conditions == nil {
		conditions = &[]configv1.ClusterOperatorStatusCondition{}
	}
	existingCondition := FindOperatorStatusCondition(*conditions, newCondition.Type)
	if existingCondition == nil {
		newCondition.LastTransitionTime = metav1.NewTime(time.Now())
		*conditions = append(*conditions, newCondition)
		return
	}
	if existingCondition.Status != newCondition.Status {
		existingCondition.Status = newCondition.Status
		existingCondition.LastTransitionTime = metav1.NewTime(time.Now())
	}
	existingCondition.Reason = newCondition.Reason
	existingCondition.Message = newCondition.Message
}
func RemoveOperatorStatusCondition(conditions *[]configv1.ClusterOperatorStatusCondition, conditionType configv1.ClusterStatusConditionType) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if conditions == nil {
		conditions = &[]configv1.ClusterOperatorStatusCondition{}
	}
	newConditions := []configv1.ClusterOperatorStatusCondition{}
	for _, condition := range *conditions {
		if condition.Type != conditionType {
			newConditions = append(newConditions, condition)
		}
	}
	*conditions = newConditions
}
func FindOperatorStatusCondition(conditions []configv1.ClusterOperatorStatusCondition, conditionType configv1.ClusterStatusConditionType) *configv1.ClusterOperatorStatusCondition {
	_logClusterCodePath()
	defer _logClusterCodePath()
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}
func IsOperatorStatusConditionTrue(conditions []configv1.ClusterOperatorStatusCondition, conditionType configv1.ClusterStatusConditionType) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return IsOperatorStatusConditionPresentAndEqual(conditions, conditionType, configv1.ConditionTrue)
}
func IsOperatorStatusConditionFalse(conditions []configv1.ClusterOperatorStatusCondition, conditionType configv1.ClusterStatusConditionType) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return IsOperatorStatusConditionPresentAndEqual(conditions, conditionType, configv1.ConditionFalse)
}
func IsOperatorStatusConditionPresentAndEqual(conditions []configv1.ClusterOperatorStatusCondition, conditionType configv1.ClusterStatusConditionType, status configv1.ConditionStatus) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Status == status
		}
	}
	return false
}
func IsOperatorStatusConditionNotIn(conditions []configv1.ClusterOperatorStatusCondition, conditionType configv1.ClusterStatusConditionType, status ...configv1.ConditionStatus) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	for _, condition := range conditions {
		if condition.Type == conditionType {
			for _, s := range status {
				if s == condition.Status {
					return false
				}
			}
			return true
		}
	}
	return true
}
