// Copyright 2024 Google, LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"fmt"
	"io"

	"github.com/h2non/filetype"

	"os"
	"os/exec"
	"strings"

	"github.com/jaycherian/gcp-go-media-search/internal/core/cor"
)

const (
	DefaultFfmpegArgs = "-analyzeduration 0 -probesize 5000000 -y -hide_banner -i %s -filter:v scale=w=%s:h=trunc(ow/a/2)*2 -f mp4 %s"
	TempFilePrefix    = "ffmpeg-output-"
	CommandSeparator  = " "
)

// FFMpegCommand is a simple command used for
// downloading a media file embedded in the message, resizing it
// and uploading the resized version to the destination bucket.
// The scale uses a dynamic scale to keep the aspect ratio of the original.
type FFMpegCommand struct {
	cor.BaseCommand
	commandPath string
	targetWidth string
}

func NewFFMpegCommand(name string, commandPath string, targetWidth string) *FFMpegCommand {
	return &FFMpegCommand{
		BaseCommand: *cor.NewBaseCommand(name),
		commandPath: commandPath,
		targetWidth: targetWidth}
}

// Execute executes the business logic of the command
func (c *FFMpegCommand) Execute(context cor.Context) {
	originalInputPath := context.Get(c.GetInputParam()).(string)

	// --- Step 1: Open the original input file ---
	originalFile, err := os.Open(originalInputPath)
	if err != nil {
		c.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(c.GetName(), fmt.Errorf("failed to open original input file: %w", err))
		return
	}
	defer originalFile.Close()

	// --- Step 2: Detect the file type to determine the correct extension ---
	// Read the first 261 bytes to get the file header for type detection.
	header := make([]byte, 261)
	if _, err := originalFile.Read(header); err != nil && err != io.EOF {
		c.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(c.GetName(), fmt.Errorf("failed to read header from input file: %w", err))
		return
	}
	// Reset the reader to the start of the file for copying later.
	originalFile.Seek(0, 0)

	kind, _ := filetype.Match(header)
	if kind == filetype.Unknown {
		// Could not determine the file type, add error or proceed with caution.
		// For this example, we'll log it and let FFmpeg try anyway.
		fmt.Println("Warning: Could not determine file type. FFmpeg might fail.")
	}

	// --- Step 3: Create a new temp input file WITH the correct extension ---
	// This solves the "No such file or directory" error from FFmpeg.
	// We create it in the current directory "." to avoid Snap permission issues with "/tmp".
	newInputFile, err := os.CreateTemp(".", "ffmpeg-input-*."+kind.Extension)
	if err != nil {
		c.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(c.GetName(), fmt.Errorf("failed to create new temp input file: %w", err))
		return
	}
	defer newInputFile.Close()
	defer os.Remove(newInputFile.Name()) // IMPORTANT: Schedule cleanup of this temp file.

	// Copy the original file's content to the new, correctly named temp file.
	if _, err := io.Copy(newInputFile, originalFile); err != nil {
		c.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(c.GetName(), fmt.Errorf("failed to copy content to new temp input: %w", err))
		return
	}
	fmt.Printf("Created new temporary input with correct extension: %s\n", newInputFile.Name())

	// --- Step 4: Create the temporary output file ---
	// Also creating in "." to avoid Snap permission issues.
	outputFile, err := os.CreateTemp(".", "ffmpeg-output-*.mp4")
	if err != nil {
		c.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(c.GetName(), fmt.Errorf("could not create a temp output file: %w", err))
		return
	}
	outputFile.Close() // Close the file so FFmpeg can write to it.

	// --- Step 5: Build and run the FFmpeg command ---
	// Using the NEW input file and the NEW output file.
	args := fmt.Sprintf(DefaultFfmpegArgs, newInputFile.Name(), c.targetWidth, outputFile.Name())
	cmd := exec.Command(c.commandPath, strings.Split(args, CommandSeparator)...)

	fmt.Printf("Executing FFmpeg command: %s\n", cmd.String())
	cmd.Stderr = os.Stderr // Pipe FFmpeg errors to standard error.

	if err := cmd.Run(); err != nil {
		os.Remove(outputFile.Name()) // Clean up the failed output file.
		c.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(c.GetName(), fmt.Errorf("error running ffmpeg: %w", err))
		return
	}

	fmt.Printf("FFmpeg processing successful. Output is at: %s\n", outputFile.Name())
	c.GetSuccessCounter().Add(context.GetContext(), 1)
	context.AddTempFile(outputFile.Name()) // Assuming this tracks files for later cleanup.
	context.Add(cor.CtxOut, outputFile.Name())
}
