package port

import "github.com/theSearchLife/videoCompressor/internal/domain"

type Reporter interface {
	JobStarted(job domain.Job)
	JobProgress(job domain.Job, progress float64)
	JobFinished(job domain.Job, err error)
	Summary(results []domain.Result)
}
