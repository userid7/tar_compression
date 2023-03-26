package audio_converter

import (
	"fmt"
	"io"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

type AudioConverter struct {
}

func NewAudioConverter() *AudioConverter {
	return &AudioConverter{}
}

func (ac *AudioConverter) ConvertWavToFlac(inputAudio io.Reader, ouputAudio io.Writer) error {
	err := ffmpeg.Input("pipe:", ffmpeg.KwArgs{"f": "wav"}).Output("pipe:", ffmpeg.KwArgs{"f": "flac"}).WithInput(inputAudio).WithOutput(ouputAudio).
		OverWriteOutput().Run()
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func (ac *AudioConverter) ConvertFlacToWav(inputAudio io.Reader, ouputAudio io.Writer) error {
	err := ffmpeg.Input("pipe:", ffmpeg.KwArgs{"f": "flac"}).Output("pipe:", ffmpeg.KwArgs{"f": "wav"}).WithInput(inputAudio).WithOutput(ouputAudio).
		OverWriteOutput().Run()
	if err != nil {
		return err
	}

	return nil
}
