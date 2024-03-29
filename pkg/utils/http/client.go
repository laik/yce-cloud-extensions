package http

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
)

var _ IClient = &client{}

type IClient interface {
	Post(url string) IClient
	Params(key string, value interface{}) IClient
	Do() error
}

func NewIClient() IClient {
	return &client{}
}

type client struct {
	url    string
	params map[string]interface{}
}

func (c *client) Post(url string) IClient {
	c.url = url
	return c
}

func (c *client) Params(key string, value interface{}) IClient {
	if c.params == nil {
		c.params = make(map[string]interface{})
	}
	c.params[key] = value
	return c
}

func (c *client) Do() error {
	body, err := json.Marshal(c.params)
	if err != nil {
		return err
	}
	response, err := resty.New().
		NewRequest().
		SetHeader("Accept", "application/json"). //default json
		SetBody(body).
		Post(c.url)
	if response != nil {
		if response.StatusCode() != 200 {
			return fmt.Errorf("post to (%s) response code (%d) error (%s) body (%s)",
				c.url,
				response.StatusCode(),
				response.String(),
				body,
			)
		}
	}
	if err != nil {
		return err
	}
	return nil

}
