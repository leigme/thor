package config

import (
	"encoding/json"
	"github.com/leigme/loki/app"
	common "github.com/leigme/thor/common"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port     int    `json:"port"`
	SavePath string `json:"savePath"`
	FileExt  string `json:"fileExt"`
	FileSize int    `json:"fileSize"`
	FileUnit int    `json:"fileUnit"`
}

var Self *Config

func Get() *Config {
	if Self == nil {
		Self = &Config{
			Port:     8080,
			SavePath: app.WorkDir(),
			FileExt:  "*",
			FileSize: 2,
			FileUnit: int(common.Mb),
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
		fileSize := os.Getenv(common.FileSize)
		if !strings.EqualFold(fileSize, "") {
			if fsi, err := strconv.Atoi(fileSize); err == nil {
				Self.FileSize = fsi
			}
		}
		fileUnit := os.Getenv(common.FileUnit)
		if !strings.EqualFold(fileUnit, "") {
			if fui, err := strconv.Atoi(fileUnit); err == nil {
				Self.FileUnit = fui
			}
		}
	}
	return Self
}

// TypeFilter upload file type filter
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
