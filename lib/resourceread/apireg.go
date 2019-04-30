package resourceread

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	apiregv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
)

var (
	apiRegScheme	= runtime.NewScheme()
	apiRegCodecs	= serializer.NewCodecFactory(apiRegScheme)
)

func init() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if err := apiregv1beta1.AddToScheme(apiRegScheme); err != nil {
		panic(err)
	}
	if err := apiregv1.AddToScheme(apiRegScheme); err != nil {
		panic(err)
	}
}
func ReadAPIServiceV1OrDie(objBytes []byte) *apiregv1.APIService {
	_logClusterCodePath()
	defer _logClusterCodePath()
	requiredObj, err := runtime.Decode(apiRegCodecs.UniversalDecoder(apiregv1.SchemeGroupVersion), objBytes)
	if err != nil {
		panic(err)
	}
	return requiredObj.(*apiregv1.APIService)
}
