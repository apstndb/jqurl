package main

import (
	"context"
	"fmt"
	"github.com/apstndb/adcplus/tokensource"
	"github.com/google/shlex"
	"github.com/jessevdk/go-flags"
	"golang.org/x/sync/errgroup"
	"io"
	"log"
	"os"
	"os/exec"
)

type opts struct {
	DataJq string `long:"data-jq"`

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

	curl := exec.Command("curl", args...)
	curl.Stderr = os.Stderr

	var injq *exec.Cmd
	var iw io.WriteCloser
	if len(iopts) > 0 {
		injq = exec.Command("gojq", iopts...)

		var ir io.ReadCloser
		ir, iw = io.Pipe()

		curl.Stdin = ir

		injq.Stdin = os.Stdin
		injq.Stdout = iw
		injq.Stderr = os.Stderr
	} else {
		curl.Stdin = os.Stdin
	}

	var outjq *exec.Cmd
	var ow io.WriteCloser
	if len(oopts) > 0 {
		outjq = exec.Command("gojq", oopts...)

		var or io.ReadCloser
		or, ow = io.Pipe()

		curl.Stdout = ow

		outjq.Stdin = or
		outjq.Stdout = os.Stdout
		outjq.Stderr = os.Stderr
	} else {
		curl.Stdout = os.Stdout
	}

	if injq != nil {
		if err := injq.Start(); err != nil {
			return fmt.Errorf("injq.Start: %w", err)
		}
	}

	if err := curl.Start(); err != nil {
		return fmt.Errorf("curl.Start: %w", err)
	}

	if outjq != nil {
		if err := outjq.Start(); err != nil {
			return fmt.Errorf("outjq.Start: %w", err)
		}
	}

	var eg errgroup.Group
	if injq != nil {
		eg.Go(func() error {
			if iw != nil {
				defer iw.Close()
			}
			if err := injq.Wait(); err != nil {
				return fmt.Errorf("injq.Wait: %w", err)
			}
			return nil
		})
	}

	eg.Go(func() error {
		if ow != nil {
			defer ow.Close()
		}
		if err := curl.Wait(); err != nil {
			return fmt.Errorf("curl.Wait: %w", err)
		}
		return nil
	})

	if outjq != nil {
		eg.Go(func() error {
			if err := outjq.Wait(); err != nil {
				return fmt.Errorf("outjq.Wait: %w", err)
			}
			return nil
		})
	}
	return eg.Wait()
}

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatalln(err)
	}
}
