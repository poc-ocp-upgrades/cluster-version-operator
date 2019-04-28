package resourcebuilder

import "k8s.io/client-go/rest"

func withProtobuf(config *rest.Config) *rest.Config {
	_logClusterCodePath()
	defer _logClusterCodePath()
	config = rest.CopyConfig(config)
	config.AcceptContentTypes = "application/vnd.kubernetes.protobuf,application/json"
	config.ContentType = "application/vnd.kubernetes.protobuf"
	return config
}
