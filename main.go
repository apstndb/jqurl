package main

import (
	"al.essio.dev/pkg/shellescape"
	"context"
	dxiter "deedles.dev/xiter"
	"fmt"
	"github.com/apstndb/adcplus/tokensource"
	"github.com/google/shlex"
	"github.com/jessevdk/go-flags"

	"iter"
	"log"
	"os/exec"
	"regexp"
	"slices"
	"strings"
)

type opts struct {
	DataJq  string `long:"data-jq"`
	Include bool   `long:"include" short:"i" description:"Include response headers in stderr"`

	IOpts string `long:"iopts"`
	OOpts string `long:"oopts"`

	IFilter string `long:"ifilter"`
	OFilter string `long:"ofilter"`

	Auth string `long:"auth"`

	DryRun bool `long:"dry-run"`

	RawInput         bool `long:"raw-input"`
	RawOutput        bool `long:"raw-output"`
	YamlInput        bool `long:"yaml-input"`
	YamlOutput       bool `long:"yaml-output"`
	ColorOutput      bool `long:"color-output"`
	MonochromeOutput bool `long:"monochrome-output"`
	CompactOutput    bool `long:"compact-output"`
	JoinOutput       bool `long:"join-output"`
	RawOutput0       bool `long:"raw-output0"`
	Slurp            bool `long:"slurp"`
}

var httpOrHTTPSRe = regexp.MustCompile("^https?://")

func joinStringSeq(i iter.Seq[string], sep string) string {
	return dxiter.Fold(i, func(l string, r string) string {
		return l + sep + r
	})
}

func run(ctx context.Context) error {
	var opts opts
	p := flags.NewParser(&opts, flags.Options(flags.IgnoreUnknown|flags.HelpFlag))
	p.Usage = "[options...] <url> [<filter>]"
	args, err := p.Parse()
	if err != nil {
		log.Fatalln(err)
	}

	iopts, err := shlex.Split(opts.IOpts)
	if err != nil {
		return err
	}

	oopts, err := shlex.Split(opts.OOpts)
	if err != nil {
		return err
	}

	var positionalOFilter string
	for i := len(args) - 1; i >= 0; i-- {
		arg := args[i]
		if httpOrHTTPSRe.MatchString(arg) {
			break
		}
		if !strings.HasPrefix(arg, "-") {
			positionalOFilter = arg
			args = slices.Concat(args[0:i], args[i+1:])
			break
		}
	}

	if positionalOFilter != "" && opts.OFilter != "" {
		return fmt.Errorf("positional filter and --ofilter are exclusive")
	}

	if positionalOFilter != "" {
		oopts = append(oopts, positionalOFilter)
	}

	if opts.DataJq != "" {
		iopts = append(iopts, "-n", opts.DataJq)
		args = append(args, "--json", "@-")
	}

	if opts.IFilter != "" {
		iopts = append(iopts, opts.IFilter)
	}

	if opts.OFilter != "" {
		oopts = append(oopts, opts.OFilter)
	}

	if opts.Include {
		args = append(args, "-D", "/dev/stderr")
	}

	if opts.CompactOutput {
		oopts = append(oopts, "-c")
	}

	if opts.RawOutput {
		oopts = append(oopts, "-r")
	}

	if opts.RawOutput0 {
		oopts = append(oopts, "--raw-output0")
	}

	if opts.ColorOutput {
		oopts = append(oopts, "-C")
	}

	if opts.MonochromeOutput {
		oopts = append(oopts, "-M")
	}

	if opts.YamlInput {
		iopts = append(iopts, "--yaml-input")
	}

	if opts.YamlOutput {
		oopts = append(oopts, "--yaml-output")
	}

	if opts.JoinOutput {
		oopts = append(oopts, "--join-output")
	}

	if opts.Slurp {
		iopts = append(iopts, "--slurp")
	}

	if opts.DryRun {
		var strs []string
		if len(iopts) > 0 {
			strs = append(strs, shellescape.QuoteCommand(slices.Concat([]string{"gojq"}, iopts)))
		}

		// authOpts needs manual quote because they don't be processed by shellescape.Quote.
		var authOpts []string
		switch opts.Auth {
		case "google":
			authOpts = append(authOpts, `-H`, `"Authorization: Bearer $(gcloud auth application-default print-access-token)"`)
		case "":
			// no action
		default:
			return fmt.Errorf("unknown --auth: %s", opts.Auth)
		}
		strs = append(strs, joinStringSeq(dxiter.Concat(dxiter.Map(slices.Values(slices.Concat([]string{"curl"}, args)), shellescape.Quote), slices.Values(authOpts)), " "))
		if len(oopts) > 0 {
			strs = append(strs, shellescape.QuoteCommand(slices.Concat([]string{"gojq"}, oopts)))
		}
		fmt.Println(strings.Join(strs, " | "))
		return nil
	}

	switch opts.Auth {
	case "google":
		ts, err := tokensource.SmartAccessTokenSource(ctx)
		if err != nil {
			return err
		}
		token, err := ts.Token()
		if err != nil {
			return err
		}
		args = append(args, "-H", fmt.Sprintf("Authorization: Bearer %s", token.AccessToken))
	case "":
		// no action
	default:
		return fmt.Errorf("unknown --auth: %s", opts.Auth)
	}

	var e Executable
	if len(iopts) > 0 {
		e = Join(e, Cmd(exec.Command("gojq", iopts...)))
	}

	e = Join(e, Cmd(exec.Command("curl", args...)))

	if len(oopts) > 0 {
		e = Join(e, Cmd(exec.Command("gojq", oopts...)))
	}

	SetAllToStd(e)

	if err = e.Start(); err != nil {
		return err
	}

	return e.Wait()
}

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatalln(err)
	}
}
