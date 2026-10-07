/*
Copyright 2024 CloudBolt, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pipes

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func editorCmd(ctx context.Context, filename string) *exec.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "notepad"
	}

	var args []string
	switch {
	case !strings.Contains(editor, " "):
		args = append(args, editor, filename)
	case !strings.Contains(editor, `"'\`):
		args = append(args, strings.Split(editor, " ")...)
		args = append(args, filename)
	default:
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "cmd"
		}
		args = append(args, shell, "/C", fmt.Sprintf("%s %q", editor, filename))
	}

	return exec.CommandContext(ctx, args[0], args[1:]...)
}
