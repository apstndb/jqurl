package main

import (
	"context"
	"deedles.dev/xiter"
	"fmt"
	"github.com/apstndb/adcplus/tokensource"
	"github.com/google/shlex"
	"github.com/jessevdk/go-flags"
	"iter"
	"log"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
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
}

var httpOrHTTPSRe = regexp.MustCompile("^https?://")

func joinStringSeq(i iter.Seq[string], sep string) string {
	return xiter.Fold(i, func(l string, r string) string {
		return l + sep + r
	})
}

func run(ctx context.Context) error {
	var opts opts
	p := flags.NewParser(&opts, flags.Options(flags.IgnoreUnknown|flags.HelpFlag))
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

	if opts.DryRun {
		var strs []string
		if len(iopts) > 0 {
			strs = append(strs, joinStringSeq(xiter.Map(xiter.Of(slices.Concat([]string{"jq"}, iopts)...), strconv.Quote), " "))
		}
		strs = append(strs, joinStringSeq(xiter.Map(xiter.Of(slices.Concat([]string{"curl"}, args)...), strconv.Quote), " "))
		if len(oopts) > 0 {
			strs = append(strs, joinStringSeq(xiter.Map(xiter.Of(slices.Concat([]string{"jq"}, oopts)...), strconv.Quote), " "))
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
