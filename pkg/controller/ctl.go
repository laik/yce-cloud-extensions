package controller

var _ Interface = &CIController{}

// Interface ....
type Interface interface {
	Run(addr string, stop <-chan struct{}) error
}
