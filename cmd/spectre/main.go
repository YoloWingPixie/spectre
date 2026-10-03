// Command spectre serves a spec folder (-dir, default the current
// directory) as a local review site: chapter pages, ID links and
// backlinks, search, open-decision answers, comments and per-item feedback
// appended to <dir>/review/*.md. Its optional audit mode builds and serves
// persistent JSON audit reports with reader decisions and notes.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/YoloWingPixie/spectre/internal/audit"
	"github.com/YoloWingPixie/spectre/internal/spec"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "spectre:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, out, diagnostics io.Writer) error {
	if len(args) > 0 {
		switch args[0] {
		case "audit":
			return audit.Run(ctx, args[1:], out, diagnostics)
		case "spec":
			args = args[1:]
		}
	}
	flags := flag.NewFlagSet("spectre", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	flags.Usage = func() {
		fmt.Fprint(diagnostics, "Usage: spectre [spec] [-dir <spec>] [-port <number>]\n       spectre audit <command> [options]\n\nSpec review is the default. Use 'spectre audit --help' for audit commands.\n\n")
		flags.PrintDefaults()
	}
	port := flags.Int("port", 8765, "port to listen on (always bound to 127.0.0.1; 0 chooses an available port)")
	dir := flags.String("dir", ".", "spec directory (review files go to <dir>/review)")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q; use 'spectre --help'", flags.Arg(0))
	}
	if *port < 0 || *port > 65535 {
		return fmt.Errorf("port must be an integer from 0 to 65535")
	}
	return serveSpec(ctx, *dir, *port, diagnostics)
}

func serveSpec(parent context.Context, dir string, port int, diagnostics io.Writer) error {
	log := slog.New(slog.NewTextHandler(diagnostics, nil))
	srv, err := spec.New(dir, log)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}
	hs := spec.NewHTTPServer(listener.Addr().String(), srv.Handler())
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := hs.Shutdown(sctx)
		if err != nil {
			err = errors.Join(err, hs.Close())
		}
		done <- err
	}()
	log.Info("spectre: serving", "url", "http://"+listener.Addr().String()+"/", "dir", dir)
	err = hs.Serve(listener)
	cancel()
	shutdownErr := <-done
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}
	return errors.Join(err, shutdownErr)
}
