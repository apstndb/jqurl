package main

import (
	"context"
	"fmt"
	"github.com/apstndb/adcplus/tokensource"
	"github.com/google/shlex"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"os/exec"
)

type opts struct {
	DataJq  string `long:"data-jq"`
	Include bool   `long:"include" short:"i" description:"Include response headers in stderr"`

	IOpts string `long:"iopts"`
	OOpts string `long:"oopts"`

	IFilter string `long:"ifilter"`
	OFilter string `long:"ofilter"`

	Auth string `long:"auth"`
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
		e = Join(e, &Cmd{Cmd: exec.Command("gojq", iopts...)})
	}

	e = Join(e, &Cmd{Cmd: exec.Command("curl", args...)})

	if len(oopts) > 0 {
		e = Join(e, &Cmd{Cmd: exec.Command("gojq", oopts...)})
	}

	e.SetStdin(os.Stdin)
	e.SetStdout(os.Stdout)
	e.SetStderr(os.Stderr)

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
