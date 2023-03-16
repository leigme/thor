package service

import (
	"errors"
	"fmt"
	"github.com/leigme/loki/file"
	"github.com/leigme/thor/common/param"
	"net/http"
	"strings"
)

type Client interface {
	Upload(filename, address string) error
}

func NewClient(opts ...ClientOption) Client {
	c := defaultClient()
	for _, apply := range opts {
		apply(&c)
	}
	return &c
}

func defaultClient() client {
	return client{
		client:    http.DefaultClient,
		paramName: string(param.File),
	}
}

type ClientOption func(*client)

func WithHttpClient(httpClient *http.Client) ClientOption {
	return func(c *client) {
		if httpClient != nil {
			c.client = httpClient
		}
	}
}

func WithParamName(name string) ClientOption {
	return func(c *client) {
		if !strings.EqualFold(name, "") {
			c.paramName = name
		}
	}
}

type client struct {
	client    *http.Client
	paramName string
}

func (c *client) Upload(filename string, address string) error {
	if !file.Exist(filename) {
		return errors.New(fmt.Sprintf("%s is not a valid file", filename))
	}
	contType, reader, err := file.CreateRequestBody(c.paramName, filename)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", address, reader)
	req.Header.Set("Content-Type", contType)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
