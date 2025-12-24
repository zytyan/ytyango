package migrate

import (
	"context"
	"flag"
	"fmt"
	g "main/globalcfg"
	"os"
	"strings"
)

type cliPaths struct {
	Main string
	Msg  string
}

type commonFlags struct {
	target     string
	db         string
	dbMain     string
	dbMsg      string
	dryRun     bool
	memoryRun  bool
	sampleRate float64
	sampleRows int
}

// RunCLI executes the migrate subcommand flow.
func RunCLI(ctx context.Context, args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}
	cmdName := strings.ToLower(args[0])
	cmdArgs := args[1:]

	switch cmdName {
	case "up":
		return handleUp(ctx, cmdArgs)
	case "down":
		return handleDown(ctx, cmdArgs)
	case "to":
		return handleTo(ctx, cmdArgs)
	case "status":
		return handleStatus(ctx, cmdArgs)
	case "help":
		printUsage()
		return nil
	default:
		printUsage()
		return fmt.Errorf("unknown migrate subcommand %q", cmdName)
	}
}

func handleUp(ctx context.Context, args []string) error {
	fs, cf := newFlagSet("up")
	to := fs.Int("to", 0, "Target version (default latest)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	targets, opts, err := resolveTargets(cf)
	if err != nil {
		return err
	}
	for _, t := range targets {
		if err := runCommand(ctx, t, Command{Type: "up", To: *to}, opts); err != nil {
			return err
		}
	}
	return nil
}

func handleDown(ctx context.Context, args []string) error {
	fs, cf := newFlagSet("down")
	to := fs.Int("to", -1, "Target version (optional)")
	step := fs.Int("step", 1, "Number of versions to roll back (default 1)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	targets, opts, err := resolveTargets(cf)
	if err != nil {
		return err
	}
	for _, t := range targets {
		if err := runCommand(ctx, t, Command{Type: "down", To: *to, Step: *step}, opts); err != nil {
			return err
		}
	}
	return nil
}

func handleTo(ctx context.Context, args []string) error {
	fs, cf := newFlagSet("to")
	to := fs.Int("to", 0, "Required target version")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *to < 0 {
		return fmt.Errorf("--to must be non-negative")
	}
	targets, opts, err := resolveTargets(cf)
	if err != nil {
		return err
	}
	for _, t := range targets {
		if err := runCommand(ctx, t, Command{Type: "to", To: *to}, opts); err != nil {
			return err
		}
	}
	return nil
}

func handleStatus(ctx context.Context, args []string) error {
	fs, cf := newFlagSet("status")
	if err := fs.Parse(args); err != nil {
		return err
	}
	targets, opts, err := resolveTargets(cf)
	if err != nil {
		return err
	}
	for _, t := range targets {
		if err := runCommand(ctx, t, Command{Type: "status"}, opts); err != nil {
			return err
		}
	}
	return nil
}

func newFlagSet(name string) (*flag.FlagSet, commonFlags) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	cf := commonFlags{}
	fs.StringVar(&cf.target, "target", "main", "Target database: main|msg|all")
	fs.StringVar(&cf.db, "db", "", "Database path override (single target only)")
	fs.StringVar(&cf.dbMain, "db-main", "", "Main database path override (when target=all)")
	fs.StringVar(&cf.dbMsg, "db-msg", "", "Msg database path override (when target=all)")
	fs.BoolVar(&cf.dryRun, "dry-run", false, "Print steps without applying")
	fs.BoolVar(&cf.memoryRun, "memory-run", false, "Run on in-memory copy with sampling; does not touch disk")
	fs.Float64Var(&cf.sampleRate, "sample-rate", 0.1, "Row sampling rate for --memory-run (0-1, default 0.1)")
	fs.IntVar(&cf.sampleRows, "sample-rows", 0, "Row sampling limit per table for --memory-run (overrides sample-rate when >0)")
	return fs, cf
}

func resolveTargets(cf commonFlags) ([]Target, ExecOptions, error) {
	target := strings.ToLower(cf.target)
	regs := registrySet()
	opts := ExecOptions{
		DryRun:     cf.dryRun,
		MemoryRun:  cf.memoryRun,
		SampleRate: cf.sampleRate,
		SampleRows: cf.sampleRows,
		Logf: func(format string, args ...any) {
			fmt.Printf(format, args...)
		},
	}
	paths := cliPaths{
		Main: g.GetConfig().DatabasePath,
		Msg:  g.GetConfig().MsgDbPath,
	}
	if cf.db != "" && target == "all" {
		return nil, opts, fmt.Errorf("--db is ambiguous for target=all; use --db-main and --db-msg")
	}

	switch target {
	case "main":
		path := cf.db
		if path == "" {
			path = paths.Main
		}
		if path == "" {
			return nil, opts, fmt.Errorf("main database path is empty; set config.yaml database-path or --db")
		}
		r := regs["main"]
		return []Target{{Registry: r, DBPath: path}}, opts, nil
	case "msg":
		path := cf.db
		if path == "" {
			path = paths.Msg
		}
		if path == "" {
			return nil, opts, fmt.Errorf("msg database path is empty; set config.yaml msg-db-path or --db")
		}
		r := regs["msg"]
		return []Target{{Registry: r, DBPath: path}}, opts, nil
	case "all":
		mainPath := cf.dbMain
		if mainPath == "" {
			mainPath = paths.Main
		}
		msgPath := cf.dbMsg
		if msgPath == "" {
			msgPath = paths.Msg
		}
		if mainPath == "" || msgPath == "" {
			return nil, opts, fmt.Errorf("database path missing: main=%q msg=%q (set config.yaml or flags)", mainPath, msgPath)
		}
		rMain := regs["main"]
		rMsg := regs["msg"]
		return []Target{
			{Registry: rMain, DBPath: mainPath},
			{Registry: rMsg, DBPath: msgPath},
		}, opts, nil
	default:
		return nil, opts, fmt.Errorf("unknown target %q (expected main|msg|all)", target)
	}
}

func printUsage() {
	fmt.Println(`Usage: ytyango migrate <command> [options]

Commands:
  up        Apply migrations up to latest or --to version
  down      Roll back migrations by --step or to --to version
  to        Migrate directly to --to version
  status    Show current version and dirty flag

Common options:
  --target main|msg|all   select database (default main)
  --db PATH               override DB path for single target
  --db-main PATH          override main DB when target=all
  --db-msg PATH           override msg DB when target=all
  --dry-run               print planned steps only
  --memory-run            run on in-memory copy (all tables) with sampling; no disk writes
  --sample-rate float     row sampling rate for memory-run (0-1, default 0.1)
  --sample-rows int       row limit per table for memory-run (overrides sample-rate)

Defaults:
  Paths load from config.yaml (globalcfg) when flags are absent.
  Bare 'ytyango migrate' shows this help.`)
}
