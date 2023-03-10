// Copyright (c) 2023, the Drone Plugins project authors.
// Please see the AUTHORS file for details. All rights reserved.
// Use of this source code is governed by an Apache 2.0 license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"os"

	"github.com/drone-plugins/drone-gh-pages/plugin"

	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
)

func main() {
	// TODO: Remove when docker runner works on Windows
	argCount := len(os.Args)
	if argCount != 1 {
		if argCount == 2 && os.Args[1] == "--help" {
			os.Exit(0)
		}

		os.Exit(1)
	}

	logrus.SetFormatter(new(formatter))

	var args plugin.Args
	if err := envconfig.Process("", &args); err != nil {
		logrus.Fatalln(err)
	}

	switch args.Level {
	case "debug":
		logrus.SetFormatter(textFormatter)
		logrus.SetLevel(logrus.DebugLevel)
	case "trace":
		logrus.SetFormatter(textFormatter)
		logrus.SetLevel(logrus.TraceLevel)
	}

	if err := plugin.Exec(context.Background(), &args); err != nil {
		logrus.Fatalln(err)
	}
}

// default formatter that writes logs without including timestamp or level information.
type formatter struct{}

func (*formatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(entry.Message), nil
}

// text formatter that writes logs with level information.
var textFormatter = &logrus.TextFormatter{
	DisableTimestamp: true,
}
