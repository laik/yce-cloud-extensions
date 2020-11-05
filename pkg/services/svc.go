package services

type IService interface {
	Start(stop <-chan struct{})
}
