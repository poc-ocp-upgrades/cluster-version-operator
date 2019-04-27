package resourceapply

import (
	"strings"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const CreateOnlyAnnotation = "release.openshift.io/create-only"

func IsCreateOnly(metadata metav1.Object) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return strings.EqualFold(metadata.GetAnnotations()[CreateOnlyAnnotation], "true")
}
