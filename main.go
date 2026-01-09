// Humantime - A command-line time tracking tool
//
// This software is a derivative work based on Zeit (https://github.com/mrusme/zeit)
// Original work copyright (c) マリウス (mrusme)
// Modifications copyright (c) Manav Panchal
//
// Licensed under the SEGV License, Version 1.0
// See LICENSE file for full license text.

package main

import (
	"os"

	"github.com/manav03panchal/humantime/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
