package stream

/*
#cgo pkg-config: libavcodec libavutil libswscale
#include <libavcodec/avcodec.h>
#include <libavutil/opt.h>
#include <libavutil/imgutils.h>
#include <libswscale/swscale.h>
#include <stdlib.h>
#include <string.h>

// Helper function for handling the AVERROR macro
static inline int get_averror_eagain() {
    return AVERROR(EAGAIN);
}

static inline int get_averror_eof() {
    return AVERROR_EOF;
}
*/
import "C"
import (
	"fmt"
	"github.com/sirupsen/logrus"
	"gocv.io/x/gocv"
	"image"
	"unsafe"
)

// FFmpegEncoder is a HEVC encoder built on FFmpeg's libavcodec
type FFmpegEncoder struct {
	config     EncoderConfig
	codec      *C.AVCodec
	codecCtx   *C.AVCodecContext
	frame      *C.AVFrame
	packet     *C.AVPacket
	swsCtx     *C.struct_SwsContext
	frameCount int64
	headers    []byte // SPS/PPS headers
	gopSize    int    // GOP size (keyframe interval)
}

func FFmpegEncoderFactory(config EncoderConfig) (*FFmpegEncoder, error) {
	encoder := &FFmpegEncoder{
		config:  config,
		gopSize: 30,
	}

	codecName := C.CString("libx265")
	defer C.free(unsafe.Pointer(codecName))

	encoder.codec = C.avcodec_find_encoder_by_name(codecName)
	if encoder.codec == nil {
		return nil, fmt.Errorf("could not find the libx265 encoder")
	}

	// Allocate the encoder context
	encoder.codecCtx = C.avcodec_alloc_context3(encoder.codec)
	if encoder.codecCtx == nil {
		return nil, fmt.Errorf("could not allocate the encoder context")
	}

	// Set the encoding parameters
	encoder.codecCtx.width = C.int(config.Width)
	encoder.codecCtx.height = C.int(config.Height)
	encoder.codecCtx.time_base = C.AVRational{num: 1, den: C.int(config.FPS)}
	encoder.codecCtx.framerate = C.AVRational{num: C.int(config.FPS), den: 1}
	encoder.codecCtx.pix_fmt = C.AV_PIX_FMT_YUV420P
	encoder.codecCtx.gop_size = C.int(encoder.gopSize)
	encoder.codecCtx.max_b_frames = 0

	if config.Bitrate > 0 {
		encoder.codecCtx.bit_rate = C.int64_t(config.Bitrate * 1000) // convert to bps
	}

	// Set the encoder preset and tune
	preset := config.Preset
	if preset == "" {
		preset = "medium"
	}
	tune := config.Tune
	if tune == "" {
		tune = "zerolatency"
	}
	defer C.free(unsafe.Pointer(C.CString(preset)))
	defer C.free(unsafe.Pointer(C.CString(tune)))

	C.av_opt_set(unsafe.Pointer(encoder.codecCtx.priv_data), C.CString("preset"), C.CString(preset), 0)
	C.av_opt_set(unsafe.Pointer(encoder.codecCtx.priv_data), C.CString("tune"), C.CString(tune), 0)

	// Add the global-header flag (used for streaming)
	encoder.codecCtx.flags |= C.AV_CODEC_FLAG_GLOBAL_HEADER

	// Open the encoder
	if C.avcodec_open2(encoder.codecCtx, encoder.codec, nil) < 0 {
		C.avcodec_free_context(&encoder.codecCtx)
		return nil, fmt.Errorf("could not open the encoder")
	}

	// Allocate the frame
	encoder.frame = C.av_frame_alloc()
	if encoder.frame == nil {
		C.avcodec_free_context(&encoder.codecCtx)
		return nil, fmt.Errorf("could not allocate the frame")
	}

	encoder.frame.format = C.int(encoder.codecCtx.pix_fmt)
	encoder.frame.width = encoder.codecCtx.width
	encoder.frame.height = encoder.codecCtx.height

	// Allocate the frame buffer
	if C.av_frame_get_buffer(encoder.frame, 0) < 0 {
		C.av_frame_free(&encoder.frame)
		C.avcodec_free_context(&encoder.codecCtx)
		return nil, fmt.Errorf("could not allocate the frame buffer")
	}

	// Allocate the packet
	encoder.packet = C.av_packet_alloc()
	if encoder.packet == nil {
		C.av_frame_free(&encoder.frame)
		C.avcodec_free_context(&encoder.codecCtx)
		return nil, fmt.Errorf("could not allocate the packet")
	}

	// Initialize the swscale context (used for BGR to YUV conversion)
	encoder.swsCtx = C.sws_getContext(
		encoder.codecCtx.width,
		encoder.codecCtx.height,
		C.AV_PIX_FMT_BGR24,
		encoder.codecCtx.width,
		encoder.codecCtx.height,
		C.AV_PIX_FMT_YUV420P,
		C.SWS_BILINEAR,
		nil, nil, nil,
	)
	if encoder.swsCtx == nil {
		C.av_packet_free(&encoder.packet)
		C.av_frame_free(&encoder.frame)
		C.avcodec_free_context(&encoder.codecCtx)
		return nil, fmt.Errorf("could not create the swscale context")
	}

	logrus.Debugf("FFmpeg encoder initialized: %dx%d @ %.2f fps, bitrate: %d kbps, preset: %s, tune: %s",
		config.Width, config.Height, config.FPS, config.Bitrate, preset, tune)

	if config.RepeatHeaders {
		if encoder.codecCtx.extradata_size > 0 {
			encoder.headers = C.GoBytes(unsafe.Pointer(encoder.codecCtx.extradata), encoder.codecCtx.extradata_size)
			logrus.Debugf("cached encoder header information, size: %d bytes", len(encoder.headers))
		}
	}

	return encoder, nil
}

// EncodeFrame encodes a single frame and returns the HEVC NAL units
func (e *FFmpegEncoder) EncodeFrame(frame gocv.Mat) ([]byte, error) {
	if frame.Empty() {
		return nil, fmt.Errorf("cannot encode an empty frame")
	}

	// Resize the frame if needed
	if frame.Cols() != e.config.Width || frame.Rows() != e.config.Height {
		resized := gocv.NewMat()
		defer resized.Close()
		gocv.Resize(frame, &resized, image.Point{X: e.config.Width, Y: e.config.Height}, 0, 0, gocv.InterpolationLinear)
		frame = resized
	}

	// Get the BGR data
	bgrData := frame.ToBytes()

	// Make sure the frame is writable
	if C.av_frame_make_writable(e.frame) < 0 {
		return nil, fmt.Errorf("could not make the frame writable")
	}

	// Copy the BGR data into C memory (avoids cgo pointer-rule problems)
	cBgrData := C.CBytes(bgrData)
	defer C.free(cBgrData)

	// Build the pointer array and linesize array in C memory
	srcData := (**C.uint8_t)(C.malloc(C.size_t(unsafe.Sizeof(uintptr(0))) * 4))
	defer C.free(unsafe.Pointer(srcData))
	srcLinesize := (*C.int)(C.malloc(C.size_t(unsafe.Sizeof(C.int(0))) * 4))
	defer C.free(unsafe.Pointer(srcLinesize))

	// Set the data and linesize for the first plane
	*srcData = (*C.uint8_t)(cBgrData)
	*srcLinesize = C.int(e.config.Width * 3)

	// BGR to YUV420P
	C.sws_scale(
		e.swsCtx,
		srcData,
		srcLinesize,
		0,
		e.codecCtx.height,
		(**C.uint8_t)(unsafe.Pointer(&e.frame.data[0])),
		(*C.int)(unsafe.Pointer(&e.frame.linesize[0])),
	)

	// Set the frame PTS
	e.frame.pts = C.int64_t(e.frameCount)
	e.frameCount++

	// Send the frame to the encoder
	ret := C.avcodec_send_frame(e.codecCtx, e.frame)
	if ret < 0 {
		return nil, fmt.Errorf("failed to send the frame to the encoder")
	}

	// Receive the encoded packet
	ret = C.avcodec_receive_packet(e.codecCtx, e.packet)
	if ret == C.get_averror_eagain() || ret == C.get_averror_eof() {
		return []byte{}, nil // needs more data, or reached the end
	} else if ret < 0 {
		return nil, fmt.Errorf("failed to receive the packet")
	}

	defer C.av_packet_unref(e.packet)

	// Copy the encoded data
	encodedData := C.GoBytes(unsafe.Pointer(e.packet.data), e.packet.size)

	// If repeated headers are configured, prepend them ahead of each keyframe
	if e.config.RepeatHeaders && len(e.headers) > 0 {
		// Check whether this is a keyframe
		isKeyframe := (e.packet.flags & C.AV_PKT_FLAG_KEY) != 0

		if isKeyframe {
			// Prepend the headers to the encoded data
			result := make([]byte, 0, len(e.headers)+len(encodedData))
			result = append(result, e.headers...)
			result = append(result, encodedData...)
			return result, nil
		}
	}

	return encodedData, nil
}

// Flush flushes the encoder buffer
func (e *FFmpegEncoder) Flush() ([]byte, error) {
	var allData []byte

	// Send a NULL frame to signal a flush
	C.avcodec_send_frame(e.codecCtx, nil)

	for {
		ret := C.avcodec_receive_packet(e.codecCtx, e.packet)
		if ret == C.get_averror_eof() || ret == C.get_averror_eagain() {
			break
		} else if ret < 0 {
			return nil, fmt.Errorf("failed to receive the packet while flushing")
		}

		encodedData := C.GoBytes(unsafe.Pointer(e.packet.data), e.packet.size)
		allData = append(allData, encodedData...)
		C.av_packet_unref(e.packet)
	}

	return allData, nil
}

// Close closes the encoder and releases its resources
func (e *FFmpegEncoder) Close() error {
	if e.swsCtx != nil {
		C.sws_freeContext(e.swsCtx)
		e.swsCtx = nil
	}

	if e.packet != nil {
		C.av_packet_free(&e.packet)
		e.packet = nil
	}

	if e.frame != nil {
		C.av_frame_free(&e.frame)
		e.frame = nil
	}

	if e.codecCtx != nil {
		C.avcodec_free_context(&e.codecCtx)
		e.codecCtx = nil
	}

	logrus.Debug("FFmpeg encoder closed")
	return nil
}

// GetHeaders returns the SPS/PPS header information (extradata)
func (e *FFmpegEncoder) GetHeaders() ([]byte, error) {
	if e.codecCtx.extradata_size > 0 {
		return C.GoBytes(unsafe.Pointer(e.codecCtx.extradata), e.codecCtx.extradata_size), nil
	}
	return []byte{}, nil
}
