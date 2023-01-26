package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	zapLog "github.com/leigme/thor/logger"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	ServerPort = "thor.server.port"
	SavePath   = "thor.save.path"
	FileExt    = "thor.file.ext"
	TypeSplit  = "|"
)

var (
	conf   *Config
	logger *zap.SugaredLogger
)

type Config struct {
	port     int
	savePath string
	fileExt  string
	exitCh   chan int
}

func init() {
	InitConfig()
	flag.IntVar(&conf.port, "p", conf.port, "web server port")
	flag.StringVar(&conf.savePath, "d", conf.savePath, "save files dir")
	flag.StringVar(&conf.fileExt, "t", conf.fileExt, "upload file ext")
}

func main() {
	flag.Parse()
	log.Printf("config: %s\n", conf.toString())
	InitLogger(conf.savePath)
	defer logger.Sync()
	r := gin.New()
	r.Use(zapLog.GinLogger(), zapLog.GinRecovery(true))
	r.GET("/running", func(c *gin.Context) {
		c.JSON(http.StatusOK, "running")
	})
	r.GET("/help", func(c *gin.Context) {
		result := make(map[string][]string, 0)
		for _, v := range r.Routes() {
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
	})
	r.POST("/upload", handlerUpload)
	s := &http.Server{Addr: fmt.Sprintf(":%d", conf.port), Handler: r}
	go func() {
		if err := s.ListenAndServe(); err != nil {
			logger.Errorf("Listen err: %s\n", err)
		}
	}()
	go gracefulExit(s)
	<-conf.exitCh
}

func gracefulExit(srv *http.Server) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGSYS, syscall.SIGTERM, os.Kill)
	sig := <-signalChan
	logger.Infof("catch signal, %+v", sig)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second) // 4秒后退出
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server Shutdown:", err)
	}
	logger.Info("server exiting")
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
	saveDir := conf.savePath
	dir := c.PostForm("dir")
	if !strings.EqualFold(dir, "") {
		saveDir = filepath.Join(saveDir, dir)
	}
	filename := filepath.Join(saveDir, f.Filename)
	err = os.MkdirAll(saveDir, os.ModePerm)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 10003, "msg": err.Error()})
		return
	}
	err = c.SaveUploadedFile(f, filename)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 10004, "msg": err.Error()})
		return
	}
	srcMd5 := c.PostForm("md5")
	if !strings.EqualFold(srcMd5, "") {
		dstMd5, err := FileMD5(filename)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 10005, "msg": err.Error()})
			return
		}
		if !strings.EqualFold(srcMd5, dstMd5) {
			errMsg := "file md5 verification fails"
			err := os.Remove(filename)
			if err != nil {
				c.JSON(http.StatusOK, gin.H{"code": 10005, "msg": errMsg + " " + err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"code": 10005, "msg": errMsg})
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

func InitConfig() {
	userHome, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("user home dir is err: %s\n", err)
	}
	sp := filepath.Join(userHome, ".thor")
	err = os.MkdirAll(sp, os.ModePerm)
	if err != nil {
		log.Fatalf("create save dir is err: %s\n", err)
	}
	fs, err := os.Stat(sp)
	if err != nil || !fs.IsDir() {
		log.Fatalf("save path is not dir: %s\n", err)
	}
	conf = &Config{
		port:     8080,
		savePath: sp,
		fileExt:  "*",
		exitCh:   make(chan int),
	}
	p := os.Getenv(ServerPort)
	if !strings.EqualFold(p, "") {
		if pi, err := strconv.Atoi(p); err == nil {
			conf.port = pi
		}
	}
	savePath := os.Getenv(SavePath)
	if !strings.EqualFold(savePath, "") {
		conf.savePath = savePath
	}
	fileExt := os.Getenv(FileExt)
	if !strings.EqualFold(fileExt, "") {
		conf.fileExt = fileExt
	}
}

func InitLogger(workPath string) {
	fs, err := os.Stat(conf.savePath)
	if err != nil || !fs.IsDir() {
		log.Fatalf("file dir: %s is err: %s\n", fs.Name(), err)
	}
	logPath := filepath.Join(workPath, "log")
	err = os.MkdirAll(logPath, os.ModePerm)
	if err != nil {
		log.Fatalf("create log dir is err: %s\n", err)
	}
	lookPath, err := exec.LookPath(os.Args[0])
	if err != nil {
		log.Fatalf("look path is err: %s\n", err)
	}
	logFile := fmt.Sprintf("%s.log", filepath.Base(lookPath))
	logger = zapLog.NewLogger(filepath.Join(logPath, logFile))
}

func (c *Config) typeFilter(fileExt string) bool {
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

func (c *Config) toString() string {
	m := make(map[string]string, 3)
	m["port"] = strconv.Itoa(c.port)
	m["savePath"] = c.savePath
	m["fileExt"] = c.fileExt
	data, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(data)
}
