package main

import (
	"errors"
	"fmt"
	"github.com/leigme/thor/config"
	"mime/multipart"
	"path"
	"strings"
)

var cfs = []checkFile{checkFileExt, checkFileSize}

type checkFile func(fileHeader *multipart.FileHeader) error

func checkFileExt(fileHeader *multipart.FileHeader) error {
	fileExt := strings.ToLower(path.Ext(fileHeader.Filename))
	if conf.TypeFilter(fileExt) {
		return nil
	}
	return errors.New(fmt.Sprintf("upload type: %s not allow", fileExt))
}

func checkFileSize(fileHeader *multipart.FileHeader) error {
	if fileHeader.Size <= config.Self.FileSize {
		return nil
	}
	return errors.New(fmt.Sprintf("upload file size is: %d than %d, please split and merge\n", fileHeader.Size, config.Self.FileSize))
}
