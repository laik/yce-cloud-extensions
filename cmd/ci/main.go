package main

import (
	ctl "github.com/laik/yce-cloud-extensions/pkg/controller"
)

func main() {
	srv := ctl.NewCIController()
	stop := make(chan struct{})
	if err := srv.Run(":8080", stop); err != nil {
		panic(err)
	}
}
