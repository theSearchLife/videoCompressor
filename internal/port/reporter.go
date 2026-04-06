package port

import "github.com/theSearchLife/videoCompressor/internal/domain"

type Reporter interface {
	JobStarted(job domain.Job)
	JobProgress(job domain.Job, progress float64)
	JobFinished(job domain.Job, result domain.Result)
	Summary(results []domain.Result)
}
