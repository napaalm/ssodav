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
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"

	"git.napaalm.xyz/napaalm/ssodav/internal/auth"
	"git.napaalm.xyz/napaalm/ssodav/internal/config"
	"git.napaalm.xyz/napaalm/ssodav/internal/url"
)

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

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
	// Check if request is restful
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		HandleRestfulLogin(w, r)
	} else {
		HandleBrowserLogin(w, r)
	}
}

func HandleBrowserLogin(w http.ResponseWriter, r *http.Request) {
	var (
		expTime time.Duration
		// Get URL to redirect to and sanitize it
		nextURL = url.SanitizeURL(r.URL.Query().Get("next"))
	)

	if r.Method == "POST" {
		// Get username and password from the form
		username := r.FormValue("username")
		password := r.FormValue("password")
		remember := r.FormValue("remember")

		// Set token/cookie expiration time
		if remember == "on" {
			expTime = 7 * 24 * time.Hour // 7 days
		} else {
			expTime = 24 * time.Hour // 1 day
		}

		// Check credentials and generate a token
		token, err := auth.AuthenticateUser(username, password, expTime)

		// Authentication failure
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Load cookie configuration
		tld := config.Config.General.TLD
		secure := config.Config.General.SecureCookies

		// Create cookie
		cookie := http.Cookie{
			Name:   "access_token",
			Value:  string(token),
			Domain: tld,
			MaxAge: int(expTime.Seconds()),
			Secure: secure,
		}
		http.SetCookie(w, &cookie)

		// Redirect after login
		if nextURL != "" {
			http.Redirect(w, r, nextURL, http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "http://"+config.Config.General.TLD, http.StatusSeeOther)
		}

		return
	}

	// Check if cookie is set
	_, err := r.Cookie("access_token")

	// If it is redirect
	if err == nil {
		if nextURL != "" {
			http.Redirect(w, r, nextURL, http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "http://"+config.Config.General.TLD, http.StatusSeeOther)
		}
		return
	}

	// Load page title from the configuration
	pageTitle := config.Config.General.PageTitle

	templates.ExecuteTemplate(w, "index.html", struct {
		PageTitle   string
		LicenseURL  string
		LicenseName string
		SourceURL   string
	}{pageTitle, licenseURL, licenseName, SourceURL})
}

func HandleRestfulLogin(w http.ResponseWriter, r *http.Request) {
	var cr credentials

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Not a POST request", http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(body, &cr); err != nil {
		http.Error(w, "Can't parse JSON", http.StatusBadRequest)
		return
	}

	// Check credentials and generate a token valid for a day
	token, err := auth.AuthenticateUser(cr.Username, cr.Password, 24*time.Hour)

	// Authentication failure
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Return token in a JSON object
	b, err := json.Marshal(map[string]string{
		"access_token": string(token),
		"type":         "bearer",
	})

	if err != nil {
		log.Println("handlers: ", err.Error())
		http.Error(w, "Error while encoding response", http.StatusInternalServerError)
	} else {
		w.Header().Add("Content-Type", "application/json")
		w.Write([]byte(b))
	}
}

// Favicon handler
func HandleFavicon(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, templatesDir+"/assets/img/favicon.ico")
}
