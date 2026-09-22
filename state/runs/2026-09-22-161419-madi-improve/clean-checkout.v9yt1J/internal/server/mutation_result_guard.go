package server

import (
	"context"
	"github.com/jackc/pgx/v5"
)

// Server-internal callbacks can record a resource ID/version and their durable
// operation checkpoint in the exact transaction that produced that result.
type mutationResultGuard func(context.Context, pgx.Tx, map[string]any) error
type mutationResultGuardKey struct{}

func withMutationResultGuard(ctx context.Context, guard mutationResultGuard) context.Context {
	if previous, ok := ctx.Value(mutationResultGuardKey{}).(mutationResultGuard); ok {
		next := guard
		guard = func(c context.Context, tx pgx.Tx, result map[string]any) error {
			if e := previous(c, tx, result); e != nil {
				return e
			}
			return next(c, tx, result)
		}
	}
	return context.WithValue(ctx, mutationResultGuardKey{}, guard)
}
