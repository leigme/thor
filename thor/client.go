package thor

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
)

type Client interface {
	Upload(filename, address string) error
}

func NewClient(c *http.Client) Client {
	return &client{
		client: c,
	}
}

type client struct {
	client *http.Client
}

func (c *client) Upload(filename string, address string) error {
	if !file.Exist(filename) {
		return errors.New(fmt.Sprintf("%s is not a valid file", filename))
	}
	contType, reader, err := createRequestBody(filename)
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

func createRequestBody(filename string) (string, io.Reader, error) {
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
	fw, err := bw.CreateFormFile(string(param.File), filepath.Base(filename))
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
