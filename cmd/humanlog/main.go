package main

import (
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/aybabtme/rgbterm"
	"github.com/casualjim/humanlog"
	"github.com/mattn/go-colorable"
	"github.com/urfave/cli"
)

var Version = "devel"

func fatalf(c *cli.Context, format string, args ...interface{}) {
	log.Printf(format, args...)
	cli.ShowAppHelp(c)
	os.Exit(1)
}

func main() {
	app := newApp()

	prefix := rgbterm.FgString(app.Name+"> ", 99, 99, 99)

	log.SetOutput(colorable.NewColorableStderr())
	log.SetFlags(0)
	log.SetPrefix(prefix)
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func newApp() *cli.App {

	skip := cli.StringSlice{}
	keep := cli.StringSlice{}

	skipFlag := &cli.StringSliceFlag{
		Name:  "skip",
		Usage: "keys to skip when parsing a log entry",
		Value: &skip,
	}

	keepFlag := &cli.StringSliceFlag{
		Name:  "keep",
		Usage: "keys to keep when parsing a log entry",
		Value: &keep,
	}

	sortLongest := &cli.BoolFlag{
		Name:        "sort-longest",
		Usage:       "sort by longest key after having sorted lexicographically",
		DefaultText: "true",
	}

	skipUnchanged := &cli.BoolFlag{
		Name:        "skip-unchanged",
		Usage:       "skip keys that have the same value than the previous entry",
		DefaultText: "true",
	}

	truncates := &cli.BoolFlag{
		Name:  "truncate",
		Usage: "truncates values that are longer than --truncate-length",
	}

	truncateLength := &cli.IntFlag{
		Name:  "truncate-length",
		Usage: "truncate values that are longer than this length",
		Value: humanlog.DefaultOptions.TruncateLength,
	}

	lightBg := &cli.BoolFlag{
		Name:    "light-bg",
		Usage:   "use black as the base foreground color (for terminals with light backgrounds)",
		EnvVars: []string{"HUMANLOG_LIGHT_BACKGROUND"},
	}

	timeFormat := &cli.StringFlag{
		Name:  "time-format",
		Usage: "output time format, see https://golang.org/pkg/time/ for details",
		Value: humanlog.DefaultOptions.TimeFormat,
	}

	ignoreInterrupts := &cli.BoolFlag{
		Name:  "ignore-interrupts, i",
		Usage: "ignore interrupts",
	}

	app := cli.NewApp()
	app.Authors = []*cli.Author{
		{Name: "Antoine Grondin", Email: "antoine@digitalocean.com"},
	}
	app.Name = "humanlog"
	app.Version = Version
	app.Usage = "reads structured logs from stdin, makes them pretty on stdout!"

	app.Flags = []cli.Flag{skipFlag, keepFlag, sortLongest, skipUnchanged, truncates, truncateLength, lightBg, timeFormat, ignoreInterrupts}

	app.Action = func(c *cli.Context) error {

		opts := humanlog.DefaultOptions

		opts.SortLongest = !c.Bool(sortLongest.Name)
		opts.SkipUnchanged = !c.Bool(skipUnchanged.Name)
		opts.Truncates = !c.Bool(truncates.Name)
		opts.TruncateLength = c.Int(truncateLength.Name)
		opts.LightBg = !c.Bool(lightBg.Name)
		opts.TimeFormat = c.String(timeFormat.Name)

		switch {
		case c.IsSet(skipFlag.Name) && c.IsSet(keepFlag.Name):
			fatalf(c, "can only use one of %q and %q", skipFlag.Name, keepFlag.Name)
		case c.IsSet(skipFlag.Name):
			opts.SetSkip(skip.Value())
		case c.IsSet(keepFlag.Name):
			opts.SetKeep(keep.Value())
		}

		if c.IsSet(strings.Split(ignoreInterrupts.Name, ",")[0]) {
			signal.Ignore(os.Interrupt)
		}

		// log.Print("reading stdin...")
		if err := humanlog.Scanner(os.Stdin, colorable.NewColorableStdout(), opts); err != nil {
			log.Fatalf("scanning caught an error: %v", err)
		}
		return nil
	}
	return app
}
