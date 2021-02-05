/*
 * url.go
 *
 * URL checking and sanitization to avoid open redirection.
 *
 * Copyright (c) 2020 Antonio Napolitano <nap@napaalm.xyz>
 *
 * This file is part of ssodav.
 *
 * ssodav is free software; you can redistribute it and/or modify it
 * under the terms of the Affero GNU General Public License as
 * published by the Free Software Foundation; either version 3, or (at
 * your option) any later version.
 *
 * ssodav is distributed in the hope that it will be useful, but WITHOUT
 * ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
 * or FITNESS FOR A PARTICULAR PURPOSE.  See the Affero GNU General
 * Public License for more details.
 *
 * You should have received a copy of the Affero GNU General Public
 * License along with ssodav; see the file LICENSE. If not see
 * <http://www.gnu.org/licenses/>.
 */

// URL checking and sanitization to avoid open redirection.
package url

import (
	"net/url"
	"strings"

	"git.napaalm.xyz/napaalm/ssodav/internal/config"
)

// Check and sanitize a redirect URL
func SanitizeURL(URL string) string {
	u, err := url.Parse(URL)

	// Unparsable URL
	if err != nil {
		return ""
	}

	// The URL must have the TLD specified in the configuration
	if !strings.HasSuffix(u.Hostname(), config.Config.General.TLD) {
		return ""
	}

	// The URL must be absolute
	if !u.IsAbs() {
		return ""
	}

	// The scheme must be http or https
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	// Santize host and scheme and return
	return u.String()
}
