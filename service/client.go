package service

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/leigme/loki/file"
	"github.com/leigme/thor/common/param"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
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
	contType, reader, err := createRequestBody(c.paramName, filename)
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

func createRequestBody(paramName, filename string) (string, io.Reader, error) {
	buf := bytes.NewBuffer(nil)
	bw := multipart.NewWriter(buf)
	f, err := os.Open(filename)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()
	if md5, err := file.Md5(filename); err == nil {
		if pw, err := bw.CreateFormField(string(param.Md5)); err == nil {
			pw.Write([]byte(md5))
		}
	}
	if strings.EqualFold(paramName, "") {
		paramName = string(param.File)
	}
	fw, err := bw.CreateFormFile(paramName, filepath.Base(filename))
	if err != nil {
		return "", nil, err
	}
	_, err = io.Copy(fw, f)
	if err != nil {
		return "", nil, err
	}
	bw.Close()
	return bw.FormDataContentType(), buf, nil
}
