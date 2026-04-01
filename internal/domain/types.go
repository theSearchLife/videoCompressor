package domain

import "time"

type Resolution string

const (
	Res720p  Resolution = "720p"
	Res1080p Resolution = "1080p"
	Res4K    Resolution = "4k"
)

func (r Resolution) Height() int {
	switch r {
	case Res720p:
		return 720
	case Res1080p:
		return 1080
	case Res4K:
		return 2160
	default:
		return 0
	}
}

type CompressionLevel string

const (
	CompressionHigh CompressionLevel = "high"
	CompressionLow  CompressionLevel = "low"
)

type Profile struct {
	Name            string
	Codec           string
	CRF             int
	Preset          string
	AudioCodec      string
	AudioBitrate    string
	ContainerFormat string
}

type VideoMeta struct {
	Path     string
	Width    int
	Height   int
	Duration time.Duration
	Codec    string
	Size     int64
}

type JobStatus string

const (
	StatusPending  JobStatus = "pending"
	StatusEncoding JobStatus = "encoding"
	StatusDone     JobStatus = "done"
	StatusFailed   JobStatus = "failed"
	StatusSkipped  JobStatus = "skipped"
)

type Job struct {
	ID         int
	Input      VideoMeta
	OutputPath string
	Profile    Profile
	Resolution Resolution
	Status     JobStatus
	Progress   float64
	Error      error
}

type Result struct {
	Job          Job
	InputSize    int64
	OutputSize   int64
	EncodeTime   time.Duration
	VMAF         float64 // 0-100, 0 means not calculated
	Error        error
}

func (r Result) Reduction() float64 {
	if r.InputSize == 0 {
		return 0
	}
	return 1 - float64(r.OutputSize)/float64(r.InputSize)
}

func (r Result) Speed(videoDuration time.Duration) float64 {
	if r.EncodeTime == 0 {
		return 0
	}
	return float64(videoDuration) / float64(r.EncodeTime)
}
