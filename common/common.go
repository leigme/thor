package common

type FU int

const Mb FU = 1024 * 1024 * 1024

const (
	ServerPort = "thor.server.port"
	SavePath   = "thor.save.path"
	FileExt    = "thor.file.ext"
	FileSize   = "thor.file.size"
	FileUnit   = "thor.file.unit"
	TypeSplit  = "|"
)

type UrlPath string

type Param string
