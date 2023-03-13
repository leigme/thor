package config

import (
	"encoding/json"
	"fmt"
	"github.com/leigme/loki/app"
	common "github.com/leigme/thor/common"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type Config struct {
	Port     string `json:"port" key:"p"`
	SaveDir  string `json:"saveDir" key:"d"`
	FileExt  string `json:"fileExt" key:"e"`
	FileSize string `json:"fileSize" key:"s"`
	FileUnit string `json:"fileUnit" key:"u"`
}

var Self *Config

func Get() *Config {
	if Self == nil {
		Self = &Config{
			Port:     "8080",
			SaveDir:  app.WorkDir(),
			FileExt:  "*",
			FileSize: "2",
			FileUnit: strconv.Itoa(int(common.Mb)),
		}
		p := os.Getenv(string(common.ServerPort))
		if !strings.EqualFold(p, "") {
			if _, err := strconv.Atoi(p); err == nil {
				Self.Port = p
			}
		}
		saveDir := os.Getenv(string(common.SaveDir))
		if !strings.EqualFold(saveDir, "") {
			Self.SaveDir = saveDir
		}
		fileExt := os.Getenv(string(common.FileExt))
		if !strings.EqualFold(fileExt, "") {
			Self.FileExt = fileExt
		}
		fileSize := os.Getenv(string(common.FileSize))
		if !strings.EqualFold(fileSize, "") {
			if _, err := strconv.Atoi(fileSize); err == nil {
				Self.FileSize = fileSize
			}
		}
		fileUnit := os.Getenv(string(common.FileUnit))
		if !strings.EqualFold(fileUnit, "") {
			if _, err := strconv.Atoi(fileUnit); err == nil {
				Self.FileUnit = fileUnit
			}
		}
	}
	return Self
}

func (c *Config) Update(src map[string]string) {
	cType := reflect.TypeOf(c).Elem()
	fmt.Println(cType.NumField())
	for i := 0; i < cType.NumField(); i++ {
		fieldTag := cType.Field(i).Tag
		if value, ok := fieldTag.Lookup("key"); ok {
			if value, ok = src[value]; ok {
				reflect.ValueOf(c).Elem().FieldByIndex([]int{i}).SetString(value)
			}
		}
	}
}

func (c *Config) ToString() string {
	data, err := json.Marshal(c)
	if err != nil {
		return "{}"
	}
	return string(data)
}
