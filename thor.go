package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/leigme/loki/file"
	"github.com/leigme/thor/config"
	"github.com/leigme/thor/logger"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var (
	conf *config.Config
	lf   *os.File
)

func init() {
	conf = config.NewConfig()
	flag.IntVar(&conf.Port, "p", conf.Port, "web server port")
	flag.StringVar(&conf.SavePath, "d", conf.SavePath, "save files dir")
	flag.StringVar(&conf.FileExt, "t", conf.FileExt, "upload file ext")
}

func main() {
	flag.Parse()
	log.Printf("config: %s\n", conf.ToString())
	var err error
	logFile := logger.NewLogger(conf.SavePath)
	lf, err = os.Open(logFile)
	if err != nil && os.IsNotExist(err) {
		lf, err = os.Create(logFile)
		if err != nil {
			log.Fatal(err)
		}
	}
	defer lf.Close()
	log.SetOutput(lf)
	go InitHttpServer(gracefulExit)
	<-conf.ExitCh
}

func gracefulExit(srv *http.Server) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, os.Kill)
	sig := <-signalChan
	log.Printf("catch signal, %+v\n", sig)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second) // 4秒后退出
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	log.Println("server exiting")
	close(conf.ExitCh)
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

func handlerUpload(c *gin.Context) {
	f, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 10001, "msg": "upload fail"})
		return
	}
	for _, cf := range cfs {
		if err = cf(f); err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 10002, "msg": err.Error()})
			return
		}
	}
	saveDir := conf.SavePath
	dir := c.PostForm("dir")
	if !strings.EqualFold(dir, "") {
		saveDir = filepath.Join(saveDir, dir)
	}
	filename := filepath.Join(saveDir, "tmp", fmt.Sprint(strings.TrimSuffix(f.Filename, filepath.Ext(f.Filename)), ".tmp"))
	err = os.MkdirAll(filepath.Dir(filename), os.ModePerm)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 10004, "msg": err.Error()})
		return
	}
	err = c.SaveUploadedFile(f, filename)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 10005, "msg": err.Error()})
		return
	}
	srcMd5 := c.PostForm("md5")
	if !strings.EqualFold(srcMd5, "") {
		tmp := filepath.Dir(filename)
		srcDir := filepath.Dir(tmp)
		dst := filepath.Join(srcDir, f.Filename)
		err = file.Merge(tmp, dst)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 10006, "msg": err.Error()})
			return
		}
		dstMd5, err := file.Md5(dst)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 10007, "msg": err.Error()})
			return
		}
		if !strings.EqualFold(srcMd5, dstMd5) {
			errMsg := "file md5 verification fails"
			err := os.Remove(filename)
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

func createTmpFilesDir(fileHeader *multipart.FileHeader) error {
	filename := filepath.Join(config.Self.SavePath, "tmp", fileHeader.Filename)
	fileDir := filepath.Dir(filename)
	return os.MkdirAll(fileDir, os.ModePerm)
}

func InitHttpServer(gracefulExit func(srv *http.Server)) {
	r := gin.New()
	r.Use(logger.GinLogger(), logger.GinRecovery(true))
	r.GET("/running", handlerRunning)
	r.GET("/help", handlerHelp(r.Routes()))
	r.POST("/upload", handlerUpload)
	s := &http.Server{Addr: fmt.Sprintf(":%d", conf.Port), Handler: r}
	go gracefulExit(s)
	if err := s.ListenAndServe(); err != nil {
		log.Fatalf("Listen err: %s\n", err)
	}
}
