package cmd

import (
	"context"
	loki "github.com/leigme/loki/cobra"
	"github.com/leigme/thor/config"
	"github.com/leigme/thor/thor"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	s := &server{
		c: config.Get(),
	}
	loki.Add(rootCmd, s,
		loki.WithFlags([]loki.Flag{
			{P: &s.c.Port, Name: "port", Shorthand: "p", Value: s.c.Port, Usage: "server port"},
			{P: &s.c.SaveDir, Name: "dir", Shorthand: "d", Value: s.c.SaveDir, Usage: "save directory"},
			{P: &s.c.FileExt, Name: "ext", Shorthand: "e", Value: s.c.FileExt, Usage: "file ext"},
			{P: &s.c.FileSize, Name: "size", Shorthand: "s", Value: s.c.FileSize, Usage: "file size"},
			{P: &s.c.FileUnit, Name: "unit", Shorthand: "u", Value: s.c.FileUnit, Usage: "file unit"},
		}),
	)
}

type server struct {
	c *config.Config
}

func (s *server) Execute() loki.Exec {
	return func(cmd *cobra.Command, args []string) {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGTERM, syscall.SIGKILL)
		ctx, cancel := context.WithCancel(context.Background())
		ts := thor.NewServer(s.c)
		ts.Start(ctx.Done())
		<-c
		cancel()
	}
}
