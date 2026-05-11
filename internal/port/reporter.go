package port

import "github.com/theSearchLife/videoCompressor/internal/domain"

type Reporter interface {
	JobStarted(job domain.Job)
	JobProgress(job domain.Job, progress float64)
	JobFinished(job domain.Job, result domain.Result)
	FileSkipped(skip domain.SkipInfo)
	Summary(results []domain.Result, skips []domain.SkipInfo)
}
