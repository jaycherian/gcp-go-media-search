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

// Package commands provides the concrete implementations of the Chain of
// Responsibility (COR) pattern's Command interface. This file defines the
// command for executing FFmpeg to transcode video files.
//
// Logic Flow:
// The `FFMpegCommand` is designed to be a step in a larger workflow. Its
// primary responsibility is to take a local video file, resize it to a
// specified width while maintaining the aspect ratio, and output a new
// video file.
//
// A key piece of logic here is the handling of temporary files. FFmpeg
// can sometimes be particular about file extensions. To avoid issues, this
// command first detects the MIME type of the input file to determine the
// correct extension. It then creates a *new* temporary file with that
// extension and copies the original content into it before passing it to
// FFmpeg. This makes the process more robust.
//
//  1. Get the path of the input file from the COR context.
//  2. Open the file and use the `filetype` library to determine its extension.
//  3. Create a new temporary input file with the correct extension (e.g., input.mp4).
//  4. Copy the contents from the original file to this new, correctly-named temp file.
//  5. Create a temporary output file.
//  6. Build and execute the `ffmpeg` command-line instruction.
//  7. If successful, add the path of the newly created (resized) video file to
//     the context so it can be used by the next command in the chain.
//  8. Track all created temporary files in the context for later cleanup.
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

// Constants used for the FFmpeg command execution.
const (
	// DefaultFfmpegArgs is a format string for the FFmpeg command.
	// -analyzeduration 0 -probesize 5000000: These flags are optimizations for faster probing of the input file.
	// -y: Overwrite output files without asking.
	// -hide_banner: Suppresses the printing of the FFmpeg banner.
	// -i %s: Specifies the input file.
	// -filter:v scale=w=%s:h=trunc(ow/a/2)*2: This is the video filter for resizing.
	//   - w=%s: Sets the target width from the command's `targetWidth` field.
	//   - h=trunc(ow/a/2)*2: Calculates the height to maintain the original aspect ratio (ow/a)
	//     and ensures the result is an even number, which is required by many codecs.
	// -f mp4 %s: Forces the output format to MP4 and specifies the output file path.
	DefaultFfmpegArgs = "-analyzeduration 0 -probesize 5000000 -y -hide_banner -i %s -filter:v scale=w=%s:h=trunc(ow/a/2)*2 -f mp4 %s"
	TempFilePrefix    = "ffmpeg-output-"
	CommandSeparator  = " "
)

// FFMpegCommand is a command implementation that wraps the execution of the FFmpeg tool.
// It is used to download a media file, resize it to a specific width while maintaining
// the aspect ratio, and prepare the resized version for the next step in a workflow.
type FFMpegCommand struct {
	cor.BaseCommand        // Embeds the BaseCommand for common functionality like naming and metrics.
	commandPath     string // The path to the FFmpeg executable (e.g., "/usr/bin/ffmpeg").
	targetWidth     string // The desired output width for the video in pixels (e.g., "240").
}

// NewFFMpegCommand is the constructor for creating a new FFMpegCommand.
//
// Inputs:
//   - name: A string name for this command instance, used for logging and telemetry.
//   - commandPath: The file system path to the FFmpeg executable.
//   - targetWidth: The target width for the video transcoding.
//
// Outputs:
//   - *FFMpegCommand: A pointer to the newly instantiated command.
func NewFFMpegCommand(name string, commandPath string, targetWidth string) *FFMpegCommand {
	return &FFMpegCommand{
		BaseCommand: *cor.NewBaseCommand(name),
		commandPath: commandPath,
		targetWidth: targetWidth}
}

// Execute contains the core logic for the command. It handles file operations,
// command building, and execution of FFmpeg.
//
// Inputs:
//   - context: The shared `cor.Context` for this workflow execution.
func (c *FFMpegCommand) Execute(context cor.Context) {
	// Retrieve the input file path from the context. This is expected to have been
	// placed here by a previous command in the chain.
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
	// Read the first 261 bytes, which is enough for the `filetype` library
	// to identify most common file formats by their magic numbers.
	header := make([]byte, 261)
	if _, err := originalFile.Read(header); err != nil && err != io.EOF {
		c.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(c.GetName(), fmt.Errorf("failed to read header from input file: %w", err))
		return
	}
	// The read moved the file pointer, so we reset it back to the beginning
	// to ensure the full file can be copied later.
	originalFile.Seek(0, 0)

	kind, _ := filetype.Match(header)
	if kind == filetype.Unknown {
		// If the file type can't be determined, we log a warning but proceed.
		// FFmpeg is often smart enough to figure it out anyway, but this is less reliable.
		fmt.Println("Warning: Could not determine file type. FFmpeg might fail.")
	}

	// --- Step 3: Create a new temp input file WITH the correct extension ---
	// This is a crucial workaround. Some tools, including FFmpeg, rely on file extensions.
	// By creating a new temp file with the detected extension (e.g., ".mp4"), we increase reliability.
	// We create it in the current directory (".") to avoid potential permission issues with /tmp,
	// especially in restricted environments like Snap packages.
	newInputFile, err := os.CreateTemp(".", "ffmpeg-input-*."+kind.Extension)
	if err != nil {
		c.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(c.GetName(), fmt.Errorf("failed to create new temp input file: %w", err))
		return
	}
	defer newInputFile.Close()
	defer os.Remove(newInputFile.Name()) // IMPORTANT: Schedule this temp file for cleanup when the function returns.

	// Copy the original file's content to the new, correctly named temp file.
	if _, err := io.Copy(newInputFile, originalFile); err != nil {
		c.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(c.GetName(), fmt.Errorf("failed to copy content to new temp input: %w", err))
		return
	}
	fmt.Printf("Created new temporary input with correct extension: %s\n", newInputFile.Name())

	// --- Step 4: Create the temporary output file ---
	// Create a placeholder file where FFmpeg will write the resized video.
	outputFile, err := os.CreateTemp(".", "ffmpeg-output-*.mp4")
	if err != nil {
		c.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(c.GetName(), fmt.Errorf("could not create a temp output file: %w", err))
		return
	}
	outputFile.Close() // Close the file handle immediately so the external FFmpeg process can write to it.

	// --- Step 5: Build and run the FFmpeg command ---
	// Populate the format string with the new temp input file, target width, and temp output file.
	args := fmt.Sprintf(DefaultFfmpegArgs, newInputFile.Name(), c.targetWidth, outputFile.Name())
	// Create the command object with the executable path and the formatted arguments.
	cmd := exec.Command(c.commandPath, strings.Split(args, CommandSeparator)...)

	fmt.Printf("Executing FFmpeg command: %s\n", cmd.String())
	cmd.Stderr = os.Stderr // Pipe FFmpeg's error output to the main application's stderr for visibility.

	// Run the command and wait for it to complete.
	if err := cmd.Run(); err != nil {
		os.Remove(outputFile.Name()) // Clean up the failed output file if the command fails.
		c.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(c.GetName(), fmt.Errorf("error running ffmpeg: %w", err))
		return
	}

	// If successful, log the output path and update metrics.
	fmt.Printf("FFmpeg processing successful. Output is at: %s\n", outputFile.Name())
	c.GetSuccessCounter().Add(context.GetContext(), 1)
	// Add the output file to the context's list of temp files for later cleanup by the chain executor.
	context.AddTempFile(outputFile.Name())
	// Add the output file path as the primary output of this command, making it available
	// as the input for the next command in the chain.
	context.Add(cor.CtxOut, outputFile.Name())
}
