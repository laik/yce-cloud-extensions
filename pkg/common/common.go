package common

import (
	"flag"
)

var (
	// InCluster Flag for the application runtime
	InCluster bool
	// DefaultConfigFile is the default bootstrap configuration
	DefaultConfigFile = "config.cfg"
)

func init() {
	flag.BoolVar(&InCluster, "incluster", false, "-incluster true")
	flag.StringVar(&DefaultConfigFile, "cfg", "./config.cfg", "-cfg ./config.cfg")
}
