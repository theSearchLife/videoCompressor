package port

import (
	"context"

	"github.com/theSearchLife/videoCompressor/internal/domain"
)

type Prober interface {
	Probe(ctx context.Context, path string) (domain.VideoMeta, error)
}
