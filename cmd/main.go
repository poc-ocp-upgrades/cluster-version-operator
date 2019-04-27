package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

const (
	componentName		= "version"
	componentNamespace	= "openshift-cluster-version"
)

var (
	rootCmd = &cobra.Command{Use: componentName, Short: "Run Cluster Version Controller", Long: ""}
)

func init() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
}
func main() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if err := rootCmd.Execute(); err != nil {
		glog.Exitf("Error executing mcc: %v", err)
	}
}
