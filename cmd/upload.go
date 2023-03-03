package cmd

import (
	"bytes"
	"fmt"
	loki "github.com/leigme/loki/cobra"
	"github.com/leigme/loki/file"
	"github.com/leigme/thor/common/param"
	"github.com/spf13/cobra"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type upload struct {
	addr   string
	workCh chan *uploadTask
	wg     sync.WaitGroup
}

func init() {
	loki.Add(rootCmd, &upload{
		addr: "http://localhost:8080",
		wg:   sync.WaitGroup{},
	})
}

func (u *upload) Execute() loki.Exec {
	return func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Fatal("Please specify upload file and server address")
		}
		filename := args[0]
		if !file.Exist(filename) {
			log.Fatal("Upload file does not exist")
		}
		httpPref := "http://"
		httpAddr := fmt.Sprint(httpPref, "localhost:8080")
		if len(args) > 1 {
			if strings.HasPrefix(args[1], "http") {
				httpAddr = args[1]
			} else {
				httpAddr = fmt.Sprint(httpPref, args[1])
			}
		}

		u.addr = httpAddr
		u.workCh = make(chan *uploadTask)

		u.doWork()
		u.doUpload(filename)
		u.wg.Wait()
	}
}

func (u *upload) doUpload(filename string) {
	fi, err := os.Stat(filename)
	if err != nil {
		log.Println(err)
		return
	}
	if fi.IsDir() {
		fis, err := os.ReadDir(filename)
		if err != nil {
			return
		}
		for _, fi := range fis {
			u.doUpload(filepath.Join(filename, fi.Name()))
		}
	} else {
		u.workCh <- &uploadTask{
			filename: filename,
			addr:     u.addr,
		}
	}
}

func (u *upload) doWork() {
	go func() {
		for ut := range u.workCh {
			go func(ut *uploadTask) {
				u.wg.Add(1)
				err := ut.doUpload()
				if err != nil {
					log.Println(err)
				}
				u.wg.Done()
			}(ut)
		}
	}()
}

type uploadTask struct {
	filename, addr string
}

func (ut *uploadTask) doUpload() error {
	time.Sleep(1 * time.Second)
	log.Printf("upload file: %s -> %s", ut.filename, ut.addr)
	//time.Sleep(2 * time.Second)
	//contType, reader, err := createRequestBody(ut.filename)
	//if err != nil {
	//	return err
	//}
	//req, err := http.NewRequest("POST", ut.addr, reader)
	//req.Header.Set("Content-Type", contType)
	//client := &http.Client{}
	//resp, err := client.Do(req)
	//if err != nil {
	//	return err
	//}
	//resp.Body.Close()
	return nil
}

func createRequestBody(filename string) (string, io.Reader, error) {
	var err error
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
