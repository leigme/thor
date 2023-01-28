package config

import (
	"encoding/json"
	"github.com/leigme/thor/common"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var Self *Config

type Config struct {
	Port     int
	SavePath string
	FileExt  string
	FileSize int64
	ExitCh   chan int
}

func NewConfig() *Config {
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
	Self = &Config{
		Port:     8080,
		SavePath: sp,
		FileExt:  "*",
		FileSize: 1024 * 1024 * 2048,
		ExitCh:   make(chan int),
	}
	p := os.Getenv(common.ServerPort)
	if !strings.EqualFold(p, "") {
		if pi, err := strconv.Atoi(p); err == nil {
			Self.Port = pi
		}
	}
	savePath := os.Getenv(common.SavePath)
	if !strings.EqualFold(savePath, "") {
		Self.SavePath = savePath
	}
	fileExt := os.Getenv(common.FileExt)
	if !strings.EqualFold(fileExt, "") {
		Self.FileExt = fileExt
	}
	return Self
}

func (c *Config) TypeFilter(fileExt string) bool {
	if strings.EqualFold(c.FileExt, "*") {
		return true
	}
	ext := strings.Split(c.FileExt, common.TypeSplit)
	for _, e := range ext {
		if strings.EqualFold(fileExt, e) {
			return true
		}
	}
	return false
}

func (c *Config) ToString() string {
	m := make(map[string]string, 3)
	m["port"] = strconv.Itoa(c.Port)
	m["savePath"] = c.SavePath
	m["fileExt"] = c.FileExt
	data, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(data)
}
