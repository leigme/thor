package cmd

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	loki "github.com/leigme/loki/cobra"
	"github.com/leigme/loki/file"
	"github.com/leigme/thor/common/param"
	"github.com/leigme/thor/common/url"
	"github.com/leigme/thor/config"
	"github.com/spf13/cobra"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	s := &server{
		c: config.Get(),
	}
	loki.Add(rootCmd, s,
		loki.WithFlags([]loki.Flag{
			{P: &s.c.Port, Name: "port", Shorthand: "p", Value: s.c.Port, Usage: "server port"},
			{P: &s.c.SaveDir, Name: "dir", Shorthand: "d", Value: s.c.SaveDir, Usage: "save directory"},
			{P: &s.c.FileExt, Name: "ext", Shorthand: "e", Value: s.c.FileExt, Usage: "file ext"},
			{P: &s.c.FileSize, Name: "size", Shorthand: "s", Value: s.c.FileSize, Usage: "file size"},
			{P: &s.c.FileUnit, Name: "unit", Shorthand: "u", Value: s.c.FileUnit, Usage: "file unit"},
		}),
	)
}

type server struct {
	c *config.Config
}

func (s *server) Execute() loki.Exec {
	return func(cmd *cobra.Command, args []string) {
		r := gin.Default()
		r.StaticFS(string(url.List), http.Dir(s.c.SaveDir))
		r.GET(string(url.Running), handlerRunning)
		r.POST(string(url.Upload), handlerUpload(s.c))
		r.GET(string(url.Help), handlerHelp(r.Routes()))
		err := r.Run(fmt.Sprintf(":%s", s.c.Port))
		if err != nil {
			log.Fatal(errors.Unwrap(err))
		}
	}
}

func handlerRunning(c *gin.Context) {
	c.JSON(http.StatusOK, "running")
}

func handlerHelp(gri gin.RoutesInfo) func(c *gin.Context) {
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

func handlerUpload(config *config.Config) func(c *gin.Context) {
	return func(c *gin.Context) {
		f, h, e := c.Request.FormFile(string(param.File))
		if e != nil {
			c.JSON(http.StatusOK, gin.H{"code": 10001, "msg": "upload fail"})
			return
		}
		if !config.TypeFilter(filepath.Ext(h.Filename)) {
			c.JSON(http.StatusOK, gin.H{"code": 10002, "msg": fmt.Sprintf("file type is not %s", config.FileExt)})
			return
		}
		if !config.SizeFilter(h.Size) {
			c.JSON(http.StatusOK, gin.H{"code": 10003, "msg": fmt.Sprintf("file size must less: %d * %d", config.FileSize, config.FileUnit)})
			return
		}
		filename := filepath.Join(config.SaveDir, h.Filename)
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
