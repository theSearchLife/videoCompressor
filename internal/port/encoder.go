package port

import (
	"context"

	"github.com/theSearchLife/videoCompressor/internal/domain"
)

type Encoder interface {
	Encode(ctx context.Context, job domain.Job, onProgress func(float64)) error
}
