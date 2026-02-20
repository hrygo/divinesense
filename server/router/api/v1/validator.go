package v1

import (
	"context"

	"github.com/hrygo/divinesense/internal/profile"
	"github.com/hrygo/divinesense/plugin/filter"
	"github.com/pkg/errors"
)

func validateSearchFilter(ctx context.Context, profile *profile.Profile, filterStr string) error {
	if filterStr == "" {
		return errors.New("filter cannot be empty")
	}

	engine, err := filter.DefaultEngine()
	if err != nil {
		return err
	}

	var dialect filter.DialectName
	switch profile.Driver {
	case "postgres":
		dialect = filter.DialectPostgres
	default:
		dialect = filter.DialectSQLite
	}

	if _, err := engine.CompileToStatement(ctx, filterStr, filter.RenderOptions{Dialect: dialect}); err != nil {
		return errors.Wrap(err, "failed to compile filter")
	}
	return nil
}
