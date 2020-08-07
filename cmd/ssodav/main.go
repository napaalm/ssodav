/*
 * main.go
 *
 * Main code.
 *
 * Copyright (c) 2020 Antonio Napolitano <nap@napaalm.xyz>
 *
 * This file is part of ssodav.
 *
 * ssodav is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.

 * ssodav is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.

 * You should have received a copy of the GNU Affero General Public
 * License along with ssodav.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"log"
	"net/http"

	"git.napaalm.xyz/napaalm/ssodav/internal/auth"
	"git.napaalm.xyz/napaalm/ssodav/internal/config"
)

// Set at compile time - see Makefile
var Version string
var SourceURL string

func main() {
	if err := config.LoadConfig("config/config.toml"); err != nil {
		panic("Errore nella lettura della configurazione!")
	}

	// Print software version
	log.Println("ssodav versione " + Version)

	// Initialize packages
	log.Println("Inizalizzazione...")
	auth.InitializeSigning()

	// Create HTTP server
	mux := http.NewServeMux()

	// File server for assets
	fs := http.FileServer(http.Dir("web/ssodav-login-page/assets"))
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))

	srvAddress := config.Config.General.Port
	srv := &http.Server{
		Addr:    srvAddress,
		Handler: mux,
	}

	log.Fatal(srv.ListenAndServe())
}