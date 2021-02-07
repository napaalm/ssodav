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
	"errors"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	text_template "text/template"
	"time"

	"git.napaalm.xyz/napaalm/ssodav/internal/auth"
	"git.napaalm.xyz/napaalm/ssodav/internal/config"
	"git.napaalm.xyz/napaalm/ssodav/internal/url"
	"golang.org/x/time/rate"
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
	loginTemplatesDir = "web/ssodav-login-page"
	openapiDir        = "web/openapi"

	// Licenza AGPL3
	licenseURL  = "https://www.gnu.org/licenses/agpl-3.0.en.html"
	licenseName = "AGPL 3.0"
)

// Viene inizializzato nel momento in cui viene importato il package
var (
	loginTemplates   = template.Must(template.ParseFiles(loginTemplatesDir + "/index.html"))
	openapiTemplates = text_template.Must(text_template.ParseFiles(openapiDir + "/openapi.yaml"))
	globalLimiter    *rate.Limiter
	accountLimiters  map[string]*rate.Limiter
	addressLimiters  map[string]*rate.Limiter
)

// Inizializza i rate limiter
func InitializeLimiters() {
	globalLimiter = rate.NewLimiter(rate.Limit(config.Config.Limits.Rate), config.Config.Limits.Burst)
	accountLimiters = make(map[string]*rate.Limiter)
	addressLimiters = make(map[string]*rate.Limiter)
}

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

// Ottiene l'indirizzo IP di una richiesta HTTP
func GetIP(r *http.Request) string {
	// Use forwarded address if present
	if fwdAddr := r.Header.Get("X-Forwarded-For"); fwdAddr != "" {
		// Get the first address in the header (see https://en.wikipedia.org/wiki/X-Forwarded-For)
		ips := strings.Split(fwdAddr, ", ")

		if len(ips) > 1 {
			return ips[0]
		}

		return fwdAddr
	}

	// Return only the ip, not the port
	return strings.Split(r.RemoteAddr, ":")[0]
}

// Controlla ed eventualmente limita le richieste
func RateLimit(username, ip string) (int, error) {
	// Create the rate limiter instances if they're not present
	if _, ok := addressLimiters[ip]; !ok {
		addressLimiters[ip] = rate.NewLimiter(rate.Every(time.Duration(3600*1000000000)), 20)
	}

	if _, ok := accountLimiters[username]; !ok {
		accountLimiters[username] = rate.NewLimiter(rate.Every(time.Duration(600*1000000000)), 5)
	}

	// Check if allowed
	globalAllow := globalLimiter.Allow()
	accountAllow := accountLimiters[username].Allow()
	addressAllow := addressLimiters[ip].Allow()

	if !globalAllow {
		return http.StatusServiceUnavailable, errors.New("Server di autenticazione non disponibile. Riprova più tardi.")
	}

	if !addressAllow {
		return http.StatusTooManyRequests, errors.New("Hai superato il numero massimo di tentativi di accesso. Riprova più tardi!")
	}

	if !accountAllow {
		return http.StatusTooManyRequests, errors.New("È stato superato il numero massimo di tentativi di accesso per questo account. Riprova più tardi!")
	}

	return http.StatusOK, nil
}

func HandleBrowserLogin(w http.ResponseWriter, r *http.Request) {
	var (
		expTime time.Duration
		// Get URL to redirect to and sanitize it
		nextURL = url.SanitizeURL(r.URL.Query().Get("next"))
	)

	if r.Method == "POST" {
		// Parse the form
		err := r.ParseForm()

		// Get username and password from the form
		username := r.PostFormValue("username")
		password := r.PostFormValue("password")
		remember := r.PostFormValue("remember")

		// Check if it is a valid request
		if err != nil || username == "" || password == "" {
			// Set status code
			w.WriteHeader(http.StatusBadRequest)

			// Load page title from the configuration
			pageTitle := config.Config.General.PageTitle

			if errT := loginTemplates.ExecuteTemplate(w, "index.html", struct {
				PageTitle    string
				LicenseURL   string
				LicenseName  string
				SourceURL    string
				Error        bool
				ErrorMessage string
			}{pageTitle, licenseURL, licenseName, SourceURL, true, "Impossibile elaborare la richiesta!"}); errT != nil {
				http.Error(w, errT.Error(), http.StatusInternalServerError)
			}

			return
		}

		// Obtain client ip address
		ip := GetIP(r)

		// Check the rate limiter
		if status, err := RateLimit(username, ip); err != nil {
			// Set status code
			w.WriteHeader(status)

			// Load page title from the configuration
			pageTitle := config.Config.General.PageTitle

			if errT := loginTemplates.ExecuteTemplate(w, "index.html", struct {
				PageTitle    string
				LicenseURL   string
				LicenseName  string
				SourceURL    string
				Error        bool
				ErrorMessage string
			}{pageTitle, licenseURL, licenseName, SourceURL, true, err.Error()}); errT != nil {
				http.Error(w, errT.Error(), http.StatusInternalServerError)
			}

			return
		}

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
			// Set 401 header
			w.WriteHeader(http.StatusUnauthorized)

			// Load page title from the configuration
			pageTitle := config.Config.General.PageTitle

			if errT := loginTemplates.ExecuteTemplate(w, "index.html", struct {
				PageTitle    string
				LicenseURL   string
				LicenseName  string
				SourceURL    string
				Error        bool
				ErrorMessage string
			}{pageTitle, licenseURL, licenseName, SourceURL, true, err.Error()}); errT != nil {
				http.Error(w, errT.Error(), http.StatusInternalServerError)
			}

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

	// Check if cookie is set and valid
	cookie, err := r.Cookie("access_token")
	if err == nil {
		err = auth.VerifyToken([]byte(cookie.Value))
	}

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

	if err := loginTemplates.ExecuteTemplate(w, "index.html", struct {
		PageTitle    string
		LicenseURL   string
		LicenseName  string
		SourceURL    string
		Error        bool
		ErrorMessage string
	}{pageTitle, licenseURL, licenseName, SourceURL, false, ""}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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

	// Get client's IP address
	ip := GetIP(r)

	// Check the rate limiter
	if status, err := RateLimit(cr.Username, ip); err != nil {
		http.Error(w, err.Error(), status)
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

// Percorso: /logout
// Endpoint di logout.
func HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Get URL to redirect to and sanitize it
	nextURL := url.SanitizeURL(r.URL.Query().Get("next"))

	// Load cookie configuration
	tld := config.Config.General.TLD
	secure := config.Config.General.SecureCookies

	// Create cookie
	cookie := http.Cookie{
		Name:    "access_token",
		Value:   "",
		Expires: time.Unix(0, 0),
		Domain:  tld,
		Secure:  secure,
	}
	http.SetCookie(w, &cookie)

	// Redirect after login
	if nextURL != "" {
		http.Redirect(w, r, nextURL, http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "http://"+config.Config.General.TLD, http.StatusSeeOther)
	}
}

// Favicon handler
func HandleFavicon(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, loginTemplatesDir+"/assets/img/favicon.ico")
}

// Swagger UI handler
func HandleSwaggerUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	http.ServeFile(w, r, openapiDir+"/index.html")
}

// openapi.yaml handler
func HandleOpenAPI(w http.ResponseWriter, r *http.Request) {
	var fqdn, url, scheme string

	w.Header().Set("Content-Type", "application/yaml")

	// Define URL
	fqdn = config.Config.General.FQDN

	if config.Config.General.SecureCookies {
		scheme = "https://"
	} else {
		scheme = "http://"
	}

	url = scheme + fqdn

	if fqdn == "localhost" {
		url += config.Config.General.Port
	}

	if err := openapiTemplates.ExecuteTemplate(w, "openapi.yaml", struct {
		Version string
		URL     string
	}{Version, url}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
