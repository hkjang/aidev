package server

import (
	"context"
	"github.com/jackc/pgx/v5"
)

// mutationGuard is installed only by trusted server workflows, never from HTTP
// input. It runs inside the existing mutation transaction immediately before
// its effect receipt/COMMIT. Callbacks must use tx only, not borrow another pool
// connection. This closes revocation races without bypassing normal REST ACLs.
type mutationGuard func(context.Context, pgx.Tx) error
type mutationGuardKey struct{}

func withMutationGuard(ctx context.Context, guard mutationGuard) context.Context {
	if previous, ok := ctx.Value(mutationGuardKey{}).(mutationGuard); ok {
		next := guard
		guard = func(ctx context.Context, tx pgx.Tx) error {
			if e := previous(ctx, tx); e != nil {
				return e
			}
			return next(ctx, tx)
		}
	}
	return context.WithValue(ctx, mutationGuardKey{}, guard)
}
