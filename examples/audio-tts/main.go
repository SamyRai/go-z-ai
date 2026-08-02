// Command audio-tts synthesizes speech from text via GLM TTS and writes the
// resulting audio bytes to a file. Pass an output path; defaults to
// ./tts-output.mp3 if omitted.
//
// Usage:
//
//	export ZAI_API_KEY=your_api_key_here
//	go run ./examples/audio-tts "Hello from go-z-ai."
package main

import (
	"context"
	"log"
	"os"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

func main() {
	text := "Hello from go-z-ai. Streaming, tools, vision — all from one Go client."
	if len(os.Args) > 1 {
		text = os.Args[1]
	}
	outPath := "tts-output.mp3"
	if len(os.Args) > 2 {
		outPath = os.Args[2]
	}

	c, err := client.NewClientFromEnv()
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	// Audio().Speech returns the raw audio bytes. The API picks the format
	// from the model — GLM-TTS ships mp3 by default; pass a voice ID via
	// the request to use a cloned voice (see voice-clone for the clone flow).
	audio, err := c.Audio().Speech(context.Background(), client.AudioSpeechRequest{
		Model: "glm-tts",
		Input: text,
	})
	if err != nil {
		log.Fatalf("speech: %v", err)
	}

	if err := os.WriteFile(outPath, audio, 0o644); err != nil {
		log.Fatalf("write: %v", err)
	}
	log.Printf("wrote %d bytes to %s", len(audio), outPath)
}
