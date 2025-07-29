// Copyright 2025 Google, LLC
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
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/jaycherian/gcp-go-media-search/internal/cloud"
	"github.com/jaycherian/gcp-go-media-search/internal/core/cor"
)

const (
	DefaultVideoDurationCmdArgs = "-v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 %s"
)

type MediaLengthCommand struct {
	cor.BaseCommand
	commandPath string
	config      *cloud.Config
}

func NewMediaLengthCommand(name string, commandPath string, outputParamName string, config *cloud.Config) *MediaLengthCommand {
	out := MediaLengthCommand{
		BaseCommand: *cor.NewBaseCommand(name),
		commandPath: commandPath,
		config:      config,
	}
	out.OutputParamName = outputParamName
	return &out
}

func (c *MediaLengthCommand) Execute(context cor.Context) {
	gcsFile := context.Get(cloud.GetGCSObjectName()).(*cloud.GCSObject)
	inputFileName := fmt.Sprintf("%s/%s/%s", c.config.Storage.GCSFuseMountPoint, gcsFile.Bucket, gcsFile.Name)

	args := fmt.Sprintf(DefaultVideoDurationCmdArgs, inputFileName)
	cmd := exec.Command(c.commandPath, strings.Split(args, CommandSeparator)...)
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		c.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(c.GetName(), fmt.Errorf("error running ffprobe: %w", err))
		return
	}

	length, err := extractVideoLengthToFullSeconds(output)
	if err != nil {
		c.GetErrorCounter().Add(context.GetContext(), 1)
		context.AddError(c.GetName(), err)
		return
	}
	c.GetSuccessCounter().Add(context.GetContext(), 1)

	context.Add(c.GetOutputParam(), length)
	context.Add(cor.CtxOut, length)
}

func extractVideoLengthToFullSeconds(output []byte) (int, error) {
	s := strings.TrimSpace(string(output))

	duration, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return int(duration) + 1, nil
	}
	return 0, fmt.Errorf("got invalid video duration: %s", s)
}
