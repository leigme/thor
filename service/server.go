package service

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/leigme/loki/app"
	"github.com/leigme/loki/file"
	"github.com/leigme/thor/common"
	"github.com/leigme/thor/common/param"
	"github.com/leigme/thor/common/url"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Server interface {
	RunningHandler() func(c *gin.Context)
	UploadHandler() func(c *gin.Context)
	HelpHandler(gri gin.RoutesInfo) func(c *gin.Context)
	Start(stopCh <-chan struct{})
}

func NewServer(opts ...ServerOption) Server {
	s := newDefaultServer()
	for _, apply := range opts {
		apply(&s)
	}
	return &s
}

func newDefaultServer() server {
	return server{
		Port:     "8080",
		SaveDir:  app.WorkDir(),
		FileExt:  "*",
		FileSize: "",
		FileUnit: "",
	}
}

type ServerOption func(server *server)

func WithPort(port int) ServerOption {
	return func(server *server) {
		if 0 < port && port < 65535 {
			server.Port = strconv.Itoa(port)
		}
	}
}

func WithSaveDir(saveDir string) ServerOption {
	return func(server *server) {
		if !strings.EqualFold(saveDir, "") {
			if fs, err := os.Stat(saveDir); err == nil && fs.IsDir() {
				server.SaveDir = saveDir
			}
		}
	}
}

func WithFileExt(fileExt string) ServerOption {
	return func(server *server) {
		if !strings.EqualFold(fileExt, "") {
			server.FileExt = fileExt
		}
	}
}

func WithFileSize(fileSize int) ServerOption {
	return func(server *server) {
		if fileSize > 0 {
			server.FileSize = strconv.Itoa(fileSize)
		}
	}
}

func WithFileUnit(fileUnit int) ServerOption {
	return func(server *server) {
		if fileUnit > 0 {
			server.FileUnit = strconv.Itoa(fileUnit)
		}
	}
}

type server struct {
	Port     string
	SaveDir  string
	FileExt  string
	FileSize string
	FileUnit string
}

func (s *server) Start(stopCh <-chan struct{}) {
	go func() {
		r := gin.Default()
		r.StaticFS(string(url.List), http.Dir(s.SaveDir))
		r.GET(string(url.Running), s.RunningHandler())
		r.POST(string(url.Upload), s.UploadHandler())
		r.GET(string(url.Help), s.HelpHandler(r.Routes()))
		err := r.Run(fmt.Sprintf(":%s", s.Port))
		if err != nil {
			log.Fatal(errors.Unwrap(err))
		}
	}()
	<-stopCh
}

func (s *server) UploadHandler() func(c *gin.Context) {
	return func(c *gin.Context) {
		f, h, e := c.Request.FormFile(string(param.File))
		if e != nil {
			c.JSON(http.StatusOK, gin.H{"code": 10001, "msg": "upload fail"})
			return
		}
		if !s.TypeFilter(filepath.Ext(h.Filename)) {
			c.JSON(http.StatusOK, gin.H{"code": 10002, "msg": fmt.Sprintf("file type is not %s", s.FileExt)})
			return
		}
		if !s.SizeFilter(h.Size) {
			c.JSON(http.StatusOK, gin.H{"code": 10003, "msg": fmt.Sprintf("file size must less: %d * %d", s.FileSize, s.FileUnit)})
			return
		}
		filename := filepath.Join(s.SaveDir, h.Filename)
		dstFile, err := os.Create(filename)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 10004, "msg": "create file failed"})
			return
		}
		_, err = io.Copy(dstFile, f)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 10005, "msg": "copy file failed"})
			return
		}
		srcMd5 := c.PostForm(string(param.Md5))
		if !strings.EqualFold(srcMd5, "") {
			dstMd5, err := file.Md5(filename)
			if err != nil {
				c.JSON(http.StatusOK, gin.H{"code": 10006, "msg": err.Error()})
				return
			}
			if !strings.EqualFold(srcMd5, dstMd5) {
				errMsg := "file md5 verification fails"
				err = os.Remove(filename)
				if err != nil {
					c.JSON(http.StatusOK, gin.H{"code": 10006, "msg": errMsg + " " + err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"code": 10006, "msg": errMsg})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"code": 10000,
			"msg":  "upload success",
			"request": gin.H{
				"save_path": filename,
			},
		})
	}
}

func (s *server) RunningHandler() func(c *gin.Context) {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, "running")
	}
}

func (s *server) HelpHandler(gri gin.RoutesInfo) func(c *gin.Context) {
	return func(c *gin.Context) {
		result := make(map[string][]string, 0)
		for _, v := range gri {
			if strings.EqualFold(v.Path, "/help") {
				continue
			}
			if result[v.Method] == nil {
				result[v.Method] = []string{v.Path}
			} else {
				result[v.Method] = append(result[v.Method], v.Path)
			}
		}
		c.JSON(http.StatusOK, result)
	}
}

// TypeFilter upload file type filter
func (s *server) TypeFilter(fileExt string) bool {
	if strings.EqualFold(s.FileExt, "*") {
		return true
	}
	ext := strings.Split(s.FileExt, common.TypeSplit)
	for _, e := range ext {
		if strings.EqualFold(fileExt, e) {
			return true
		}
	}
	return false
}

func (s *server) SizeFilter(fileSize int64) bool {
	fs, _ := strconv.Atoi(s.FileSize)
	fu, _ := strconv.Atoi(s.FileUnit)
	return int64(fs*fu) >= fileSize
}
