// Copyright (c) 2023, the Drone Plugins project authors.
// Please see the AUTHORS file for details. All rights reserved.
// Use of this source code is governed by an Apache 2.0 license that can be
// found in the LICENSE file.

package plugin

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

//nolint:errcheck
func writeCard(path string, card interface{}) {
	data, _ := json.Marshal(card)

	switch {
	case path == "/dev/stdout":
		writeCardTo(os.Stdout, data)
	case path == "/dev/stderr":
		writeCardTo(os.Stderr, data)
	case path != "":
		os.WriteFile(path, data, 0o644) //nolint:gomnd,gosec
	}
}

//nolint:errcheck
func writeCardTo(out io.Writer, data []byte) {
	encoded := base64.StdEncoding.EncodeToString(data)

	io.WriteString(out, "\u001B]1338;")
	io.WriteString(out, encoded)
	io.WriteString(out, "\u001B]0m")
	io.WriteString(out, "\n")
}

func contents(str string) (string, error) {
	// Check for the empty string
	if str == "" {
		return str, nil
	}

	isFilePath := false

	// See if the string is referencing a URL
	if u, err := url.Parse(str); err == nil {
		switch u.Scheme {
		case "http", "https":
			res, err := http.Get(str)
			if err != nil {
				return "", err
			}

			defer res.Body.Close()
			b, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return "", fmt.Errorf("could not read response: %w", err)
			}

			return string(b), nil

		case "file":
			// Fall through to file loading
			str = u.Path
			isFilePath = true
		}
	}

	// See if the string is referencing a file
	_, err := os.Stat(str)
	if err == nil {
		b, err := os.ReadFile(str)
		if err != nil {
			return "", fmt.Errorf("could not load file %s: %w", str, err)
		}

		return string(b), nil
	}

	if isFilePath {
		return "", fmt.Errorf("could not load file %s: %w", str, err)
	}

	// Its a regular string
	return str, nil
}
