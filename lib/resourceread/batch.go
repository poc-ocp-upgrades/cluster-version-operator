package resourceread

import (
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	batchScheme	= runtime.NewScheme()
	batchCodecs	= serializer.NewCodecFactory(batchScheme)
)

func init() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if err := batchv1.AddToScheme(batchScheme); err != nil {
		panic(err)
	}
}
func ReadJobV1OrDie(objBytes []byte) *batchv1.Job {
	_logClusterCodePath()
	defer _logClusterCodePath()
	requiredObj, err := runtime.Decode(batchCodecs.UniversalDecoder(batchv1.SchemeGroupVersion), objBytes)
	if err != nil {
		panic(err)
	}
	return requiredObj.(*batchv1.Job)
}
