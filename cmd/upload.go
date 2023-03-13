package cmd

import (
	"fmt"
	loki "github.com/leigme/loki/cobra"
	"github.com/leigme/thor/thor"
	"github.com/spf13/cobra"
	"log"
	"net/http"
	"strings"
)

var (
	filename, addr string
)

func init() {
	u := &upload{uploader: thor.NewUploader(http.DefaultClient)}
	loki.Add(rootCmd, u, loki.WithFlags([]loki.Flag{
		{P: &filename, Name: "filename", Shorthand: "f", Usage: "filename"},
		{P: &addr, Name: "address", Shorthand: "a", Usage: "address"},
	}))
}

type upload struct {
	uploader thor.Uploader
}

func (u *upload) Execute() loki.Exec {
	return func(cmd *cobra.Command, args []string) {
		if !strings.HasPrefix(addr, "http") {
			addr = fmt.Sprintf("http://%s", addr)
		}
		err := u.uploader.Upload(filename, addr)
		if err != nil {
			log.Fatal(err)
		}
	}
}
