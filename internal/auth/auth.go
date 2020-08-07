/*
 * auth.go
 *
 * Funzione per autenticare un utente.
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

// Package per l'autenticazione degli utenti.
package auth

import (
	"fmt"
	"time"

	"git.napaalm.xyz/napaalm/ssodav/internal/config"
	"github.com/gbrlsnchs/jwt/v3"
	ldap "github.com/go-ldap/ldap/v3"
)

var (
	jwtSigner *jwt.HMACSHA
)

// Errore di autenticazione
type AuthenticationError struct {
	user string
}

func (e *AuthenticationError) Error() string {
	return fmt.Sprintf("Authentication error for user %s.", e)
}

// Errore di generazione del token
type JWTCreationError struct {
	username string
}

func (e *JWTCreationError) Error() string {
	return fmt.Sprintf("Failed to sign the JWT token for username %s.", e.username)
}

// Valore di ritorno di ParseToken
type UserInfo struct {
	Username string `json:"username"`
	IsAdmin  bool   `json:"isAdmin"`
}

// Formato del payload JWT
type customPayload struct {
	Payload jwt.Payload
}

// Inizializza l'algoritmo per la firma HS256
func InitializeSigning() {

	// Ottiene la chiave segreta dalla configurazione
	secret := config.Config.General.JWTSecret

	// Inizializza l'algoritmo
	jwtSigner = jwt.NewHS256([]byte(secret))
}

// Controlla le credenziali sul server LDAP
func checkCredentials(username string, password string) error {

	// Utente readonly per la ricerca dell'utente effettivo
	bindusername := "readonly"
	bindpassword := "password"

	// Ottiene l'indirizzo del server dalla configurazione
	host := config.Config.LDAP.URI
	port := config.Config.LDAP.Port

	// Connessione al server LDAP
	l, err := ldap.DialURL("ldap://" + host + ":" + port)
	if err != nil {
		return &AuthenticationError{username}
	}
	defer l.Close()

	// Per prima cosa effettuo l'accesso con un utente readonly
	err = l.Bind(bindusername, bindpassword)
	if err != nil {
		return &AuthenticationError{username}
	}

	// Cerco l'username richiesto
	searchRequest := ldap.NewSearchRequest(
		"dc=example,dc=com",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectClass=organizationalPerson)(uid=%s))", username),
		[]string{"dn"},
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		return &AuthenticationError{username}
	}

	// Verifico il numero di utenti corrispondenti e ottendo il DN dell'utente
	if len(sr.Entries) != 1 {
		return &AuthenticationError{username}
	}

	userdn := sr.Entries[0].DN

	// Verifica la password
	err = l.Bind(userdn, password)
	if err != nil {
		return &AuthenticationError{username}
	}

	return nil
}

// Genera un token
func getToken(username string, isAdmin bool) ([]byte, error) {

	var (
		// Ottiene il tempo corrente
		now = time.Now()

		// Carico i domini autorizzati dalla configurazione
		domains = config.Config.General.Domains

		// Carico il FQDN
		fqdn = config.Config.General.FQDN

		// Inizializzo l'audience
		aud = jwt.Audience{}
	)

	// Definisco l'audience
	for _, domain := range domains {
		aud = append(aud, "http://"+domain)
		aud = append(aud, "https://"+domain)
	}

	// Definisco il payload
	pl := customPayload{
		Payload: jwt.Payload{
			Issuer:         fqdn,
			Subject:        username,
			Audience:       aud,
			ExpirationTime: jwt.NumericDate(now.Add(24 * time.Hour)),
			IssuedAt:       jwt.NumericDate(now),
		},
	}

	// Firma il token
	token, err := jwt.Sign(pl, jwtSigner)

	if err != nil {
		return nil, &JWTCreationError{username}
	}

	return token, nil
}
