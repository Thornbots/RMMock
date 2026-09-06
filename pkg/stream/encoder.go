package stream

import "gocv.io/x/gocv"

// EncoderConfig describes the encoder configuration
type EncoderConfig struct {
	Width         int
	Height        int
	FPS           float64
	Bitrate       int    // kbps
	Preset        string // https://trac.ffmpeg.org/wiki/Encode/H.265#ConstantRateFactorCRF
	Tune          string // https://trac.ffmpeg.org/wiki/Encode/H.265#ConstantRateFactorCRF
	RepeatHeaders bool   // whether to resend the SPS/PPS headers before every frame
}

type Encoder interface {
	EncodeFrame(frame gocv.Mat) ([]byte, error)
	Flush() ([]byte, error)
	Close() error
	GetHeaders() ([]byte, error)
}
