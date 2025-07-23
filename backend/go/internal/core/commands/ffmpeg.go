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

	"os"
	"os/exec"
	"strings"

	"github.com/jaycherian/gcp-go-media-search/internal/cloud"
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
	config          *cloud.Config
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
func NewFFMpegCommand(name string, commandPath string, targetWidth string, config *cloud.Config) *FFMpegCommand {
	return &FFMpegCommand{
		BaseCommand: *cor.NewBaseCommand(name),
		commandPath: commandPath,
		targetWidth: targetWidth,
		config:      config}
}

// Execute contains the core logic for the command. It handles file operations,
// command building, and execution of FFmpeg.
//
// Inputs:
//   - context: The shared `cor.Context` for this workflow execution.
func (c *FFMpegCommand) Execute(context cor.Context) {
	msg := context.Get(c.GetInputParam()).(*cloud.GCSObject)
	inputFileName := fmt.Sprintf("%s/%s/%s", c.config.Storage.GCSFuseMountPoint, msg.Bucket, msg.Name)

	file, err := os.Open(inputFileName)
	if err != nil {
		c.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(c.GetName(), err)
		return
	}
	tempFile, _ := os.CreateTemp("", TempFilePrefix)

	args := fmt.Sprintf(DefaultFfmpegArgs, file.Name(), c.targetWidth, tempFile.Name())
	cmd := exec.Command(c.commandPath, strings.Split(args, CommandSeparator)...)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		c.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(c.GetName(), fmt.Errorf("error running ffmpeg: %w", err))
		return
	}
	outputFile := fmt.Sprintf("%s/%s/%s", c.config.Storage.GCSFuseMountPoint, c.config.Storage.LowResOutputBucket, msg.Name)

	MoveFile(tempFile.Name(), outputFile)
	c.GetSuccessCounter().Add(context.GetContext(), 1)
	context.AddTempFile(outputFile)
	context.Add(cor.CtxOut, outputFile)
}

func MoveFile(sourcePath, destPath string) error {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("could not open source file: %v", err)
	}
	defer inputFile.Close()

	outputFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("could not open dest file: %v", err)
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, inputFile)
	if err != nil {
		return fmt.Errorf("could not copy to dest from source: %v", err)
	}

	inputFile.Close()

	err = os.Remove(sourcePath)
	if err != nil {
		return fmt.Errorf("could not remove source file: %v", err)
	}
	return nil
}
