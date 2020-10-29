package http

import "testing"

var _ IClient = &FakeClient{}

func (f *FakeClient) Post(url string) IClient {
	f.method = "post"
	f.url = url
	return f
}
func (f *FakeClient) Params(key string, value interface{}) IClient {
	f.params[key] = value
	return f
}
func (f *FakeClient) Do() error {
	return nil
}

type FakeClient struct {
	url    string
	method string
	params map[string]interface{}
}

func TestClient(t *testing.T) {
	var client IClient = &FakeClient{
		params: make(map[string]interface{}),
	}

	if err := client.Post(":8081").
		Params("123", "xx").
		Params("abc", "yy").
		Do(); err != nil {
		t.Fatal("non expect error")
	}

	if client.(*FakeClient).url != ":8081" {
		t.Fatal("non expect error")
	}

}
