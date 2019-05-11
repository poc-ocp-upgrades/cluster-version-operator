package resourceread

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	coreScheme	= runtime.NewScheme()
	coreCodecs	= serializer.NewCodecFactory(coreScheme)
)

func init() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if err := corev1.AddToScheme(coreScheme); err != nil {
		panic(err)
	}
}
func ReadConfigMapV1OrDie(objBytes []byte) *corev1.ConfigMap {
	_logClusterCodePath()
	defer _logClusterCodePath()
	requiredObj, err := runtime.Decode(coreCodecs.UniversalDecoder(corev1.SchemeGroupVersion), objBytes)
	if err != nil {
		panic(err)
	}
	return requiredObj.(*corev1.ConfigMap)
}
func ReadServiceAccountV1OrDie(objBytes []byte) *corev1.ServiceAccount {
	_logClusterCodePath()
	defer _logClusterCodePath()
	requiredObj, err := runtime.Decode(coreCodecs.UniversalDecoder(corev1.SchemeGroupVersion), objBytes)
	if err != nil {
		panic(err)
	}
	return requiredObj.(*corev1.ServiceAccount)
}
func ReadNamespaceV1OrDie(objBytes []byte) *corev1.Namespace {
	_logClusterCodePath()
	defer _logClusterCodePath()
	requiredObj, err := runtime.Decode(coreCodecs.UniversalDecoder(corev1.SchemeGroupVersion), objBytes)
	if err != nil {
		panic(err)
	}
	return requiredObj.(*corev1.Namespace)
}
func ReadServiceV1OrDie(objBytes []byte) *corev1.Service {
	_logClusterCodePath()
	defer _logClusterCodePath()
	requiredObj, err := runtime.Decode(coreCodecs.UniversalDecoder(corev1.SchemeGroupVersion), objBytes)
	if err != nil {
		panic(err)
	}
	return requiredObj.(*corev1.Service)
}
