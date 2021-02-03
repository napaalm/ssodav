/*
 * auth_test.go
 *
 * File di test per il package auth.
 *
 * Copyright (c) 2021 Antonio Napolitano <nap@napaalm.xyz>
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

package auth

import (
	"fmt"
	"testing"

	"git.napaalm.xyz/napaalm/ssodav/internal/config"
)

func TestAll(t *testing.T) {
	// Carica e visualizza la configurazione di test
	config.LoadConfig("./config_test.toml")
	fmt.Println(config.Config)

	// Genera un token
	InitializeSigning()
	token, err := AuthenticateUser("professor", "professor", 10000000)

	if err == nil {
		t.Log(string(token))
	} else {
		t.Error(err)
		t.FailNow()
	}

	// Verifica il token generato
	err = VerifyToken(token)

	if err != nil {
		t.Error(err)
	}

	err = VerifyToken([]byte("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJQYXlsb2FkIjp7ImlzcyI6InNzb2RhdiIsInN1YiI6ImFkbWluIiwiYXVkIjpbImh0dHA6Ly9leGFtcGxlLm9yZyIsImh0dHBzOi8vZXhhbXBsZS5vcmciLCJodHRwOi8vdGVzdC5leGFtcGxlLm9yZyIsImh0dHBzOi8vdGVzdC5leGFtcGxlLm9yZyJdLCJleHAiOjM1OTM1NDA0MDksImlhdCI6MTU5MzU0MDQwOX0sImlzQWRtaW4iOmZhbHNlfQ.GWUw0j5S5x0DfcL4hDbP9B9vcEoKoYMWGXb1Y5J0yFQ"))

	if err != nil {
		t.Error(err)
	}
}
