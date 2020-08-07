/*
 * handlers.go
 *
 * Package per gestire le diverse pagine ed i relativi template.
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

// Package per gestire le diverse pagine ed i relativi template.
package handlers

import (
	"encoding/json"
	"git.napaalm.xyz/napaalm/ssodav/internal/auth"
	"git.napaalm.xyz/napaalm/ssodav/internal/config"
	"net/http"
	"text/template"
)

var (
	Version   string
	SourceURL string
)

const (
	templatesDir = "web/ssodav-login-page"

	// Licenza AGPL3
	licenseURL  = "https://www.gnu.org/licenses/agpl-3.0.en.html"
	licenseName = "AGPL 3.0"
)

// Viene inizializzato nel momento in cui viene importato il package
var templates = template.Must(template.ParseFiles(templatesDir + "/index.html"))

// Handler per qualunque percorso diverso da tutti gli altri percorsi riconosciuti.
// Caso particolare è la homepage (/); per ogni altro restituisce 404.
func HandleRootOr404(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	HandleLogin(w, r)
}

// Percorso: /
// Pagina di accesso.
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Ottiene username e password
		r.ParseForm()
		username_list, ok0 := r.Form["username"]
		password_list, ok1 := r.Form["password"]
		if !ok0 || !ok1 || len(username_list) != 1 || len(password_list) != 1 {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		username := username_list[0]
		password := password_list[0]

		// Controlla le credenziali e ottiene il token
		token, err := auth.AuthenticateUser(username, password)

		// Se l'autenticazione fallisce ritorna 401
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Ottiene l'header Accept
		acceptHeader := r.Header.Get("Accept")

		// Se la richiesta non viene da un browser...
		if acceptHeader == "text/json" {
			// Ritorna il token in una risposta JSON
			b, err := json.Marshal(map[string]string{
				"access_token": string(token),
				"type":         "bearer",
			})

			if err != nil {
				// Internal Server Error
			} else {
				w.Write([]byte(b))
			}
		} else {
			// Ottiene la configurazione per i cookie
			tld := config.Config.General.TLD
			secure := config.Config.General.SecureCookies

			// Crea e imposta il cookie
			cookie := http.Cookie{
				Name:   "access_token",
				Value:  string(token),
				Domain: tld,
				MaxAge: 86400, // 24 ore
				Secure: secure,
			}
			http.SetCookie(w, &cookie)

			// Reindirizza a /
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}

	// Ottiene il cookie
	_, err := r.Cookie("access_token")

	// Se riesce ad ottenerlo reindirizza a /
	if err == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Carica il titolo della pagina dalla configurazione
	pageTitle := config.Config.General.PageTitle

	templates.ExecuteTemplate(w, "index.html", struct {
		PageTitle   string
		LicenseURL  string
		LicenseName string
		SourceURL   string
	}{pageTitle, licenseURL, licenseName, SourceURL})
}
