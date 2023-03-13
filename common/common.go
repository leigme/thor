package common

type FUT int

type CAT string

const Mb FUT = 1024 * 1024 * 1024

const (
	ServerPort CAT = "thor.server.port"
	SaveDir    CAT = "thor.save.dir"
	FileExt    CAT = "thor.file.ext"
	FileSize   CAT = "thor.file.size"
	FileUnit   CAT = "thor.file.unit"
)

const TypeSplit = "|"

type UrlPath string

type Param string
