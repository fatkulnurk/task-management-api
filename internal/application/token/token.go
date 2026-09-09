package token

import "context"

type TokenService interface {
	Issue(ctx context.Context, userID string) (string, int64, error)
	Verify(ctx context.Context, raw string) (string, error)
}
