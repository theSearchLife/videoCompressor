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

type CompressionStrategy string

const (
	StrategyQuality      CompressionStrategy = "quality"
	StrategyBalanced     CompressionStrategy = "balanced"
	StrategySizePriority CompressionStrategy = "size"
)

type AudioMode string

const (
	AudioKeep   AudioMode = "keep"
	AudioLow    AudioMode = "low"
	AudioMedium AudioMode = "medium"
	AudioHigh   AudioMode = "high"
)

type Profile struct {
	Name            string
	Codec           string
	CRF             int
	Preset          string
	AudioCodec      string
	AudioBitrate    string
	ContainerFormat string
	FrameRate       int // 0 = keep original
}

type VideoMeta struct {
	Path           string
	Width          int
	Height         int
	Duration       time.Duration
	Codec          string
	AudioCodec     string // ffprobe codec_name for the first audio stream, "" if none
	Size           int64
	FrameRate      float64 // source fps from ffprobe
	PixelFormat    string
	BitDepth       int
	SLog3          bool
	SLog3Detection string
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
	Job        Job
	InputSize  int64
	OutputSize int64
	EncodeTime time.Duration
	VMAF       float64 // 0-100, 0 means not calculated
	Error      error
}

// SkipCode is a short identifier used by the reporter to render the
// fixed-width skip-reason slot in the unified log row.
type SkipCode string

const (
	SkipCodePrevCompress  SkipCode = "prev_compress"
	SkipCodeAlreadyDone   SkipCode = "already_done"
	SkipCodePathCollision SkipCode = "path_collision"
	SkipCodeUncompressed  SkipCode = "uncompreseable"
)

// SkipInfo records an input file that was not encoded, with the reason.
// Skipped files still appear in the per-file output stream and in the
// final summary so every input is accounted for. RowID is the input's
// position in scan order; the reporter prints it in the same `[N]` slot
// used by encoded jobs so every input has a stable visible row id.
type SkipInfo struct {
	RowID  int
	Path   string
	Size   int64
	Code   SkipCode
	Reason string
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
