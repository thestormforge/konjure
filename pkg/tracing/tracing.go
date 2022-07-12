/*
Copyright 2022 GramLabs, Inc.

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

package tracing

import (
	"os/exec"
	"syscall"
	"time"

	"github.com/rs/zerolog"
)

// Log is the debug/trace log used by Konjure. It is initially set to a
// disabled logger but may be globally overridden.
var Log = zerolog.Nop()

// Exec logs a trace message with request/response fields for the specified
// command (which should already have run when this is called).
func Exec(cmd *exec.Cmd, start time.Time) {
	Log.Trace().
		Func(func(e *zerolog.Event) {
			req := zerolog.Dict()
			if cmd != nil {
				args := zerolog.Arr()
				for _, arg := range cmd.Args {
					args.Str(arg)
				}
				req = req.
					Str("path", cmd.Path).
					Array("args", args).
					Str("dir", cmd.Dir)
			}
			e.Dict("execRequest", req)
		}).
		Func(func(e *zerolog.Event) {
			resp := zerolog.Dict()
			if cmd.ProcessState != nil {
				resp.Int("pid", cmd.ProcessState.Pid()).
					Dur("totalTime", time.Since(start)).
					Dur("userTime", cmd.ProcessState.UserTime()).
					Dur("systemTime", cmd.ProcessState.SystemTime())
				if ws, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
					switch {
					case ws.Exited():
						resp = resp.Int("exitStatus", ws.ExitStatus())
					case ws.Signaled():
						resp = resp.Stringer("signal", ws.Signal())
					case ws.Stopped():
						if ws.StopSignal() == syscall.SIGTRAP && ws.TrapCause() != 0 {
							resp = resp.Int("trapCause", ws.TrapCause())
						} else {
							resp = resp.Stringer("stopSignal", ws.StopSignal())
						}
					case ws.Continued():
						resp = resp.Bool("continued", true)
					}
					if ws.CoreDump() {
						resp = resp.Bool("coreDumped", true)
					}
				} else {
					resp = resp.Int("exitCode", cmd.ProcessState.ExitCode())
				}
			}
			e.Dict("execResponse", resp)
		}).
		Msgf("Exit Code: %d", cmd.ProcessState.ExitCode())
}
