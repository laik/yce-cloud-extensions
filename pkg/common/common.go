package common

import (
	"flag"
)

var (
	// INCLUSTER Flag for the application runtime
	INCLUSTER bool
)

func init() {
	flag.BoolVar(&INCLUSTER, "in_cluster", false, "-in_cluster true")
}
