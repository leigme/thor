package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"runtime/debug"
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
	conf *Config
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
	var (
		lf  *os.File
		err error
	)
	logFile := NewLogger(conf.savePath)
	lf, err = os.Open(logFile)
	if err != nil && os.IsNotExist(err) {
		lf, err = os.Create(logFile)
		if err != nil {
			log.Fatal(err)
		}
	}
	defer lf.Close()
	log.SetOutput(lf)
	r := gin.New()
	r.Use(ginLogger(), ginRecovery(true))
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
		if err = s.ListenAndServe(); err != nil {
			log.Fatalf("Listen err: %s\n", err)
		}
	}()
	go gracefulExit(s)
	<-conf.exitCh
}

func gracefulExit(srv *http.Server) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGSYS, syscall.SIGTERM, os.Kill)
	sig := <-signalChan
	log.Printf("catch signal, %+v\n", sig)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second) // 4秒后退出
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	log.Println("server exiting")
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

func NewLogger(workPath string) string {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Lmicroseconds)
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
	return filepath.Join(conf.savePath, "log", logFile)
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

// GinLogger 接收gin框架默认的日志
func ginLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()

		cost := time.Since(start)
		log.Println(path,
			fmt.Sprint("status", c.Writer.Status()),
			fmt.Sprint("method", c.Request.Method),
			fmt.Sprint("path", path),
			fmt.Sprint("query", query),
			fmt.Sprint("ip", c.ClientIP()),
			fmt.Sprint("user-agent", c.Request.UserAgent()),
			fmt.Sprint("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()),
			fmt.Sprint("cost", cost),
		)
	}
}

// GinRecovery recover掉项目可能出现的panic，并使用zap记录相关日志
func ginRecovery(stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				if brokenPipe {
					log.Println(c.Request.URL.Path,
						fmt.Sprint("error", err),
						fmt.Sprint("request", string(httpRequest)),
					)
					// If the connection is dead, we can't write a status to it.
					c.Error(err.(error)) // nolint: errcheck
					c.Abort()
					return
				}

				if stack {
					log.Println("[Recovery from panic]",
						fmt.Sprint("error", err),
						fmt.Sprint("request", string(httpRequest)),
						fmt.Sprint("stack", string(debug.Stack())),
					)
				} else {
					log.Println("[Recovery from panic]",
						fmt.Sprint("error", err),
						fmt.Sprint("request", string(httpRequest)),
					)
				}
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
