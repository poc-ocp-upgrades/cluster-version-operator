package resourceread

import (
	imagev1 "github.com/openshift/api/image/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	imageScheme	= runtime.NewScheme()
	imageCodecs	= serializer.NewCodecFactory(imageScheme)
)

func init() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if err := imagev1.AddToScheme(imageScheme); err != nil {
		panic(err)
	}
}
func ReadImageStreamV1(objBytes []byte) (*imagev1.ImageStream, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	requiredObj, err := runtime.Decode(imageCodecs.UniversalDecoder(imagev1.SchemeGroupVersion), objBytes)
	if err != nil {
		return nil, err
	}
	return requiredObj.(*imagev1.ImageStream), nil
}
