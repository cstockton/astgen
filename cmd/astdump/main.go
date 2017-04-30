package main

import (
	"flag"
	"fmt"
	"go/format"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cstockton/astgen/astfrom"
	goon "github.com/shurcooL/go-goon"
)

const (
	flagFormatUsage = "providing the -f flag also prints formatted text"
	flagHelpUsage   = "display usage information and exit"
	helpText        = `
astdump is a simple utility to print ast related information for Go source. It
simply constructs an AST and dumps it using "github.com/shurcooL/go-goon".

  *Warning* Do not use this utility with >15 lines, the output is very verbose.

For more information please see:

  https://github.com/cstockton/astgen

Example:

  # Dump an arbitrary expression:
  astdump '_ *= 123'

  # Each argument is pasred separately:
  astdump someIdent "_ *= 123" 'someVar := "somestr"'

  # Dump a small chunk of source from stdin.
  cat source.go | astdump -

  # Dump and reformat the source text with -f
  cat source.go | astdump -f -

Usage:

  astdump [flags...] [source...]

Flags:
`
)

// flags
var (
	flagHelp   bool
	flagFormat bool
)

var (
	stdinNotice sync.Once
	stdinReads  int64
)

func init() {
	flag.BoolVar(&flagHelp, "h", false, flagHelpUsage)
	flag.BoolVar(&flagFormat, "fmt", false, flagFormatUsage)
	flag.BoolVar(&flagFormat, "f", false, flagFormatUsage+` [short]`)
}

func doStdinNotice() {
	stdinNotice.Do(func() {
		go func() {
			select {
			case <-time.After(time.Second / 2):
				if atomic.LoadInt64(&stdinReads) != 1 {
					switch len(flag.Args()) {
					case 0:
						fmt.Fprintln(os.Stderr, `no args given, waiting for stdin...`)
					default:
						fmt.Fprintln(os.Stderr, `dash arg given, waiting for stdin...`)
					}
				}
			}
		}()
	})
}

func getStdinArg() string {
	if atomic.LoadInt64(&stdinReads) > 0 {
		exit(1, `attempt to perform multiple reads from stdin`)
	}
	doStdinNotice()
	b, err := ioutil.ReadAll(io.LimitReader(os.Stdin, 1e6))
	atomic.AddInt64(&stdinReads, 1)
	must(err)
	return string(b)
}

func getArgs() []string {
	args := flag.Args()
	if len(args) == 0 {
		args = append(args, `-`)
	}
	for idx, arg := range args {
		if arg == `-` {
			args[idx] = getStdinArg()
		}
	}
	return args
}

func main() {
	flag.Parse()
	if flagHelp {
		fmt.Println(strings.TrimSpace(helpText))
		flag.PrintDefaults()
		os.Exit(0)
	}

	args := getArgs()
	for idx, arg := range args {
		node := astfrom.Source(arg)

		fmt.Printf("  --------  [Source - Arg #%v]  --------\n", idx)
		goon.Dump(node)

		if flagFormat {
			fmt.Printf("\n  --------  [Formatted - Arg #%v]  --------\n", idx)
			fset := token.NewFileSet()
			err := format.Node(os.Stdout, fset, node)
			must(err)
			fmt.Printf("\n\n")
		}
	}
}

func mutlExcl(bools ...bool) bool {
	count := 0
	for _, b := range bools {
		if b {
			count++
		}
	}
	return count > 1
}

func must(err error) {
	if err == nil {
		return
	}
	exit(0, "exiting due to err:", err)
}

func exit(code int, msg string, a ...interface{}) {
	if code == 0 {
		fmt.Fprintf(os.Stdout, msg+"\n", a...)
	} else {
		fmt.Fprintf(os.Stderr, msg+"\n", a...)
	}
	os.Exit(code)
}
