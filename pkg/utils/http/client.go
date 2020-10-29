package http

type IClient interface {
	Post(url string) IClient
	Params(key string, value interface{}) IClient
	Do() error
}

func NewIClient() IClient {
	return nil
}
