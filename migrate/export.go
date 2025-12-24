package migrate

import (
	"context"
	"database/sql"

	"main/migrate/core"
)

type (
	Step        = core.Step
	Migration   = core.Migration
	Registry    = core.Registry
	Target      = core.Target
	ExecOptions = core.ExecOptions
	Command     = core.Command
)

var ErrDirtyDatabase = core.ErrDirtyDatabase

func RunCommand(ctx context.Context, target Target, cmd Command, opts ExecOptions) error {
	return core.RunCommand(ctx, target, cmd, opts)
}

func CheckVersion(ctx context.Context, db *sql.DB, expected int, name string) error {
	return core.CheckVersion(ctx, db, expected, name)
}

func registrySet() map[string]Registry {
	m := core.RegistrySet(MigrationsMain, ExpectedSchemaVersionMain, MigrationsMsg, ExpectedSchemaVersionMsg)
	out := make(map[string]Registry, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
