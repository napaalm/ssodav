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
	"errors"
	"fmt"
	"log"
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
	username string
}

func (e *AuthenticationError) Error() string {
	return fmt.Sprintf("Errore di autenticazione oppure utente \"%s\" non esistente.", e.username)
}

// Errore di generazione del token
type JWTCreationError struct {
	username string
}

func (e *JWTCreationError) Error() string {
	return fmt.Sprintf("Failed to sign the JWT token for username \"%s\".", e.username)
}

// Valore di ritorno di ParseToken
type UserInfo struct {
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Group    string `json:"group"`
}

var dummyUserInfo = UserInfo{
	"h4x0r",
	"1337 h4x0r",
	"1337",
}

// Formato del payload JWT
type customPayload struct {
	Payload  jwt.Payload
	FullName string `json:"full_name"`
	Group    string `json:"group"`
}

// Inizializza l'algoritmo per la firma HS256
func InitializeSigning() {

	// Ottiene la chiave segreta dalla configurazione
	secret := config.Config.General.JWTSecret

	// Inizializza l'algoritmo
	jwtSigner = jwt.NewHS256([]byte(secret))
}

// Verifica le credenziali, ottiene il livello di permessi dell'utente e restituisce il token.
func AuthenticateUser(username, password string, exp time.Duration) ([]byte, error) {
	var (
		err      error = nil
		userInfo       = UserInfo{username, "unknown", "unknown"}
	)

	if !config.Config.General.DummyAuth {
		// Controlla le credenziali
		userInfo, err = checkCredentials(username, password)
	}

	if err == nil {
		// Genera il token
		token, err := getToken(userInfo, exp)

		if err != nil {
			return nil, err
		}

		return token, nil

	} else {
		return nil, err
	}
}

// Controlla le credenziali sul server LDAP
func checkCredentials(username string, password string) (UserInfo, error) {

	// Ottiene la configurazione
	host := config.Config.LDAP.URI
	port := config.Config.LDAP.Port
	baseDN := config.Config.LDAP.BaseDN
	bindUserDN := "cn=" + config.Config.LDAP.Username + "," + baseDN
	bindPassword := config.Config.LDAP.Password

	// Connessione al server LDAP
	l, err := ldap.DialURL("ldap://" + host + ":" + port)
	if err != nil {
		log.Println("auth: ", err.Error())
		return dummyUserInfo, &AuthenticationError{username}
	}
	defer l.Close()

	// Per prima cosa effettuo l'accesso con un utente admin
	err = l.Bind(bindUserDN, bindPassword)
	if err != nil {
		log.Println("auth: ", err.Error())
		return dummyUserInfo, &AuthenticationError{username}
	}

	// Cerco l'username richiesto
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(uid=%s)", ldap.EscapeFilter(username)), // Escape username
		[]string{"dn", "cn", "ou"},
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		log.Println("auth: ", err.Error())
		return dummyUserInfo, &AuthenticationError{username}
	}

	// Verifico il numero di utenti corrispondenti e ottendo il DN dell'utente
	if len(sr.Entries) != 1 {
		return dummyUserInfo, &AuthenticationError{username}
	}

	userDN := sr.Entries[0].DN
	fullName := sr.Entries[0].GetAttributeValue("cn")
	group := sr.Entries[0].GetAttributeValue("ou")

	// Verifica la password
	err = l.Bind(userDN, password)
	if err != nil {
		return dummyUserInfo, errors.New("Password errata!")
	}

	return UserInfo{username, fullName, group}, nil
}

// Genera un token
func getToken(userInfo UserInfo, exp time.Duration) ([]byte, error) {

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
			Subject:        userInfo.Username,
			Audience:       aud,
			ExpirationTime: jwt.NumericDate(now.Add(exp)),
			IssuedAt:       jwt.NumericDate(now),
		},
		FullName: userInfo.FullName,
		Group:    userInfo.Group,
	}

	// Firma il token
	token, err := jwt.Sign(pl, jwtSigner)

	if err != nil {
		return nil, &JWTCreationError{userInfo.Username}
	}

	return token, nil
}

// Verify a token
func VerifyToken(token []byte) error {

	var (
		// Ottiene il tempo corrente
		now = time.Now()

		// Carico i domini autorizzati dalla configurazione
		domains = config.Config.General.Domains

		// Inizializzo l'audience
		aud = jwt.Audience{}
	)

	// Definisco l'audience
	for _, domain := range domains {
		aud = append(aud, "http://"+domain)
		aud = append(aud, "https://"+domain)
	}

	var (
		// Inizializzo i "validatori"
		iatValidator = jwt.IssuedAtValidator(now)
		expValidator = jwt.ExpirationTimeValidator(now)
		audValidator = jwt.AudienceValidator(aud)

		// Costruisco il validatore supremo
		pl              customPayload
		validatePayload = jwt.ValidatePayload(&pl.Payload, iatValidator, expValidator, audValidator)
	)

	// Verifico il token
	_, err := jwt.Verify(token, jwtSigner, &pl, validatePayload)

	if err != nil {
		return err
	}

	return nil
}
