/*
Copyright 2021 GramLabs, Inc.

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

package main

import (
	"context"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/thestormforge/konjure/internal/command"
)

var (
	version = ""
	commit  = "HEAD"
	date    = time.Now().String()
)

func init() {
	cobra.EnableCommandSorting = false
}

func main() {
	// TODO Wrap `http.DefaultTransport` so it includes the UA string

	ctx := context.Background()
	cmd := command.NewRootCommand(version, commit, date)
	if err := cmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
