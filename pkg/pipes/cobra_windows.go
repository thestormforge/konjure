package pipes

import (
	"context"
	"os"
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

	return exec.CommandContext(cmd.Context(), args[0], args[1:]...)
}
