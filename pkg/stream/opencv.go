package stream

import (
	"gocv.io/x/gocv"
	"log"
)

type OpenCVCaptureParams struct {
	frameWidth  int
	frameHeight int
	fps         float64
}

func GetOpenCVCaptureParam(capture *gocv.VideoCapture) OpenCVCaptureParams {

	return OpenCVCaptureParams{
		frameWidth:  int(capture.Get(gocv.VideoCaptureFrameWidth)),
		frameHeight: int(capture.Get(gocv.VideoCaptureFrameHeight)),
		fps:         capture.Get(gocv.VideoCaptureFPS),
	}
}
func GetOpencvVideoStream[T interface{ int | string }](source T) *gocv.VideoCapture {
	stream, err := gocv.OpenVideoCapture(source)
	if err != nil {
		log.Fatalf("failed to open video stream: %v", err)
	}
	return stream
}
