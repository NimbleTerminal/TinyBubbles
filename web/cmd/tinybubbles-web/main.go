package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"nimbleterminal.dev/tinybubbles/web/internal/scaffold"
)

func main() {
	force := flag.Bool("force", false, "replace generated files and prune obsolete assets")
	assets := flag.Bool("assets", false, "build page, Booba assets, and license notices")
	module := flag.String("module", "", "create a Go module for a new app")
	local := flag.String("local-harness", "", "local harness module directory for --module")
	version := flag.String("harness-version", "", "published harness version for --module")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: tinybubbles-web [--force] [--assets] directory")
		os.Exit(2)
	}
	if *module != "" {
		if *assets || strings.ContainsAny(*module, " \t\r\n") || (*local == "" && *version == "") {
			fmt.Fprintln(os.Stderr, "--module requires --local-harness or --harness-version and cannot use --assets")
			os.Exit(2)
		}
		v := *version
		replacement := ""
		if *local != "" {
			absolute, err := filepath.Abs(*local)
			if err != nil {
				panic(err)
			}
			v = "v0.0.0"
			replacement = fmt.Sprintf("\nreplace nimbleterminal.dev/tinybubbles/web => %q\n", absolute)
		}
		data := fmt.Sprintf(`module %s

go 1.26.8

require (
 nimbleterminal.dev/tinybubbles/web %s
 github.com/NimbleMarkets/go-booba v0.9.1
)

replace charm.land/bubbletea/v2 => github.com/neomantra/bubbletea/v2 v2.0.0-20261009201241-c397c32a5074
%s
tool (
 nimbleterminal.dev/tinybubbles/web/cmd/tinybubbles-web
 github.com/NimbleMarkets/go-booba/cmd/booba-assets
)
`, *module, v, replacement)
		if err := scaffold.WriteModule(flag.Arg(0), []byte(data)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if err := run(flag.Arg(0), *force, *assets); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(target string, force, assets bool) error {
	if !assets {
		return scaffold.Write(target, force, false)
	}
	stage, err := os.MkdirTemp("", "tinybubbles-web-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	if err := scaffold.Write(stage, true, true); err != nil {
		return err
	}
	for _, args := range [][]string{{"go", "tool", "booba-assets", "--no-index", stage}, {"python3", filepath.Join(stage, "notices.py"), stage}} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return publishAssets(stage, target, force)
}
