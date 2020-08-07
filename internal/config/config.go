/*
 * config.go
 *
 * File per il caricamento e gestione della configurazione
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

// File per il caricamento e gestione della configurazione
package config

import (
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type config struct {
	General general `toml:"Generale"`
	LDAP    ldap    `toml:"LDAP"`
}

type general struct {
	FQDN          string   `toml:"fqdn_sito"`
	Domains       []string `toml:"domini_autorizzati"`
	Port          string   `toml:"porta_http"`
	JWTSecret     string   `toml:"chiave_firma"`
	SecureCookies bool     `toml:"cookie_sicuri"`
	DummyAuth     bool     `toml:"dummy_auth"`
}

type ldap struct {
	URI  string `toml:"uri"`
	Port string `toml:"porta"`
}

var Config config

func LoadConfig(path string) error {
	absPath, err := filepath.Abs(path)

	if err != nil {
		return err
	}

	if _, err := toml.DecodeFile(absPath, &Config); err != nil {
		return err
	}

	return nil
}
