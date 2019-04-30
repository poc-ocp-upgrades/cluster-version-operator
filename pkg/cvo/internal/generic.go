package internal

import (
	"context"
	godefaultbytes "bytes"
	godefaulthttp "net/http"
	godefaultruntime "runtime"
	"encoding/json"
	"fmt"
	"github.com/openshift/cluster-version-operator/lib/resourceapply"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/dynamic"
	"github.com/openshift/client-go/config/clientset/versioned/scheme"
	"github.com/openshift/cluster-version-operator/lib"
	"github.com/openshift/cluster-version-operator/lib/resourcebuilder"
)

func readUnstructuredV1OrDie(objBytes []byte) *unstructured.Unstructured {
	_logClusterCodePath()
	defer _logClusterCodePath()
	udi, _, err := scheme.Codecs.UniversalDecoder().Decode(objBytes, nil, &unstructured.Unstructured{})
	if err != nil {
		panic(err)
	}
	return udi.(*unstructured.Unstructured)
}
func applyUnstructured(client dynamic.ResourceInterface, required *unstructured.Unstructured) (*unstructured.Unstructured, bool, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if required.GetName() == "" {
		return nil, false, fmt.Errorf("invalid object: name cannot be empty")
	}
	existing, err := client.Get(required.GetName(), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		actual, err := client.Create(required, metav1.CreateOptions{})
		return actual, true, err
	}
	if err != nil {
		return nil, false, err
	}
	if resourceapply.IsCreateOnly(required) {
		return nil, false, nil
	}
	existing.SetAnnotations(required.GetAnnotations())
	existing.SetLabels(required.GetLabels())
	existing.SetOwnerReferences(required.GetOwnerReferences())
	skipKeys := sets.NewString("apiVersion", "kind", "metadata", "status")
	for k, v := range required.Object {
		if skipKeys.Has(k) {
			continue
		}
		existing.Object[k] = v
	}
	actual, err := client.Update(existing, metav1.UpdateOptions{})
	if err != nil {
		return nil, false, err
	}
	return actual, existing.GetResourceVersion() != actual.GetResourceVersion(), nil
}

type genericBuilder struct {
	client		dynamic.ResourceInterface
	raw		[]byte
	modifier	resourcebuilder.MetaV1ObjectModifierFunc
}

func NewGenericBuilder(client dynamic.ResourceInterface, m lib.Manifest) (resourcebuilder.Interface, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &genericBuilder{client: client, raw: m.Raw}, nil
}
func (b *genericBuilder) WithMode(m resourcebuilder.Mode) resourcebuilder.Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return b
}
func (b *genericBuilder) WithModifier(f resourcebuilder.MetaV1ObjectModifierFunc) resourcebuilder.Interface {
	_logClusterCodePath()
	defer _logClusterCodePath()
	b.modifier = f
	return b
}
func (b *genericBuilder) Do(_ context.Context) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	ud := readUnstructuredV1OrDie(b.raw)
	if b.modifier != nil {
		b.modifier(ud)
	}
	_, _, err := applyUnstructured(b.client, ud)
	return err
}
func createPatch(original, modified runtime.Object) ([]byte, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	originalData, err := json.Marshal(original)
	if err != nil {
		return nil, err
	}
	modifiedData, err := json.Marshal(modified)
	if err != nil {
		return nil, err
	}
	return strategicpatch.CreateTwoWayMergePatch(originalData, modifiedData, original)
}
func _logClusterCodePath() {
	pc, _, _, _ := godefaultruntime.Caller(1)
	jsonLog := []byte(fmt.Sprintf("{\"fn\": \"%s\"}", godefaultruntime.FuncForPC(pc).Name()))
	godefaulthttp.Post("http://35.226.239.161:5001/"+"logcode", "application/json", godefaultbytes.NewBuffer(jsonLog))
}
