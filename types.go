// Copyright 2017 Alfred Landrum. All rights reserved.
// Use of this source code is governed by the license
// found in the LICENSE.txt file.

package fromenv

import (
	"flag"
	"net/url"
)

// URL merely provides a net/url.URL wrapper that matches the flag.Value
// interface for ease of using with fromenv.
type URL url.URL

// Set calls url.ParseRequestURI on the input string; on success, this
// instance is set to the parsed net/url.URL result.
func (u *URL) Set(s string) error {
	nu, err := url.ParseRequestURI(s)
	if err != nil {
		return err
	}
	*u = URL(*nu)
	return nil
}

func (u *URL) String() string {
	return (*url.URL)(u).String()
}

var _ flag.Value = (*URL)(nil)
