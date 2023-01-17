package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const TypeSplit = "|"

var (
	conf *config
)

type config struct {
	port     int
	savePath string
	fileExt  string
	exitCh   chan int
}

func init() {
	conf = &config{
		exitCh: make(chan int),
	}
	flag.IntVar(&conf.port, "p", 8080, "web service port")
	flag.StringVar(&conf.savePath, "d", "", "save files dir")
	flag.StringVar(&conf.fileExt, "t", "*", "upload file ext")
	flag.Parse()
	if strings.EqualFold(conf.savePath, "") {
		userHome, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("user home dir is err: %s\n", err)
		}
		conf.savePath = filepath.Join(userHome, ".thor")
		err = os.MkdirAll(conf.savePath, os.ModePerm)
		if err != nil {
			log.Fatalf("create save dir is err: %s\n", err)
		}
	}
	fs, err := os.Stat(conf.savePath)
	if err != nil || !fs.IsDir() {
		log.Fatalf("file dir is err: %s\n", err)
	}
}

func main() {
	r := gin.Default()
	r.GET("/running", func(c *gin.Context) {
		c.JSON(http.StatusOK, "running")
	})
	r.POST("/upload", handlerUpload)
	s := &http.Server{Addr: fmt.Sprintf(":%d", conf.port), Handler: r}
	go func() {
		if err := s.ListenAndServe(); err != nil {
			log.Printf("Listen err: %s\n", err)
		}
	}()
	go gracefulExit(s)
	<-conf.exitCh
}

func gracefulExit(srv *http.Server) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGSYS, syscall.SIGTERM, os.Kill)
	sig := <-signalChan
	log.Printf("catch signal, %+v", sig)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second) // 4秒后退出
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	log.Printf("server exiting")
	close(conf.exitCh)
}

func handlerUpload(c *gin.Context) {
	f, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 10001, "msg": "upload fail"})
		return
	}
	fileExt := strings.ToLower(path.Ext(f.Filename))
	if !conf.typeFilter(fileExt) {
		c.JSON(http.StatusOK, gin.H{"code": 10002, "msg": fmt.Sprintf("upload type: %s not allow", fileExt)})
		return
	}
	filename := filepath.Join(conf.savePath, f.Filename)
	err = c.SaveUploadedFile(f, filename)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 10003, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 10000,
		"msg":  "upload success",
		"request": gin.H{
			"save_path": filename,
		},
	})
}

func (c *config) typeFilter(fileExt string) bool {
	if strings.EqualFold(c.fileExt, "*") {
		return true
	}
	ext := strings.Split(c.fileExt, TypeSplit)
	for _, e := range ext {
		if strings.EqualFold(fileExt, e) {
			return true
		}
	}
	return false
}
