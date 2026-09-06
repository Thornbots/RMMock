package stream

import (
	"github.com/sirupsen/logrus"
	"gocv.io/x/gocv"
	"os"
	"sync"
)

// StartEncodedStream starts an HEVC-encoded video stream
func StartEncodedStream[T interface{ int | string }](source T, wg *sync.WaitGroup) {
	defer wg.Done()

	conn := GetUDPConn()
	logrus.Info("connected to UDP server")
	defer conn.Close()

	stream := GetOpencvVideoStream(source)
	defer stream.Close()
	streamParam := GetOpenCVCaptureParam(stream)
	logrus.Debugf("video resolution: %dx%d, FPS: %.2f", streamParam.frameWidth, streamParam.frameHeight, streamParam.fps)

	// create the HEVC encoder
	encoder, err := FFmpegEncoderFactory(EncoderConfig{
		Width:         streamParam.frameWidth,
		Height:        streamParam.frameHeight,
		FPS:           streamParam.fps,
		Bitrate:       2000,
		Preset:        "veryfast",
		Tune:          "zerolatency",
		RepeatHeaders: true,
	})
	if err != nil {
		logrus.Fatalf("failed to create encoder: %v", err)
	}
	defer encoder.Close()

	// fetch and send the SPS/PPS headers
	headers, err := encoder.GetHeaders()
	if err != nil {
		logrus.Warnf("failed to fetch encoder headers: %v", err)
	} else if len(headers) > 0 {
		logrus.Debugf("sent encoder headers (%d bytes)", len(headers))
	}

	// create a Mat to hold the frame
	frame := gocv.NewMat()
	defer frame.Close()

	// The preview window is created only once (upstream recreates it every frame inside the loop, which leaks and causes the process to exit).
	// Set the nowindow=1 environment variable to disable the preview entirely, for headless runs.
	var window *gocv.Window
	if os.Getenv("nowindow") != "1" {
		window = gocv.NewWindow("Encoded Camera Feed")
		defer window.Close()
	}

	frameID := uint16(0)
	for {
		if ok := stream.Read(&frame); !ok {
			logrus.Error("failed to read camera frame")
			break
		}

		if frame.Empty() {
			logrus.Warn("empty frame")
			continue
		}

		// encode the frame
		encodedData, err := encoder.EncodeFrame(frame)
		if err != nil {
			logrus.Fatalf("encoding failed: %v", err)
			continue
		}

		// skip if the encoder is buffering
		if len(encodedData) == 0 {
			continue
		}

		logrus.Debugf("frame %d encoding done, size: %d bytes (original: %d bytes)",
			frameID, len(encodedData), len(frame.ToBytes()))

		SendPacket(conn, encodedData, frameID)
		frameID = (frameID + 1) % 65535

		if window != nil {
			window.IMShow(frame)
			if window.WaitKey(1) >= 0 {
				break
			}
		}
	}

	// flush the encoder
	logrus.Debug("flushing encoder buffer")
	flushedData, _ := encoder.Flush()
	if len(flushedData) > 0 {
		conn.Write(flushedData)
		logrus.Debug("sent flush data: %d bytes", len(flushedData))
	}
	logrus.Debug("encoded stream finished")
}
