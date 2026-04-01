package port

import "context"

type Scanner interface {
	Scan(ctx context.Context, root string) ([]string, error)
}
