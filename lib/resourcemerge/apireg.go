package resourcemerge

import (
	"k8s.io/apimachinery/pkg/api/equality"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
)

func EnsureAPIService(modified *bool, existing *apiregv1.APIService, required apiregv1.APIService) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	EnsureObjectMeta(modified, &existing.ObjectMeta, required.ObjectMeta)
	if !equality.Semantic.DeepEqual(existing.Spec, required.Spec) {
		*modified = true
		existing.Spec = required.Spec
	}
}
