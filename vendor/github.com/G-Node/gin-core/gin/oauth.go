// Copyright (c) 2016, German Neuroinformatics Node (G-Node)
//
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted under the terms of the BSD License. See
// LICENSE file in the root of the Project.

package gin

import "time"

// AuthResponse is used by the auth server to serve authentication responses.
type AuthResponse struct {
	Scope       string `json:"scope"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// TokenResponse is used by the auth server to serve token request responses.
type TokenResponse struct {
	TokenType    string  `json:"token_type"`
	Scope        string  `json:"scope"`
	AccessToken  string  `json:"access_token"`
	RefreshToken *string `json:"refresh_token"`
}

// TokenInfo is used by the auth server to serve token information requests.
type TokenInfo struct {
	URL        string    `json:"url"`
	JTI        string    `json:"jti"`
	EXP        time.Time `json:"exp"`
	ISS        string    `json:"iss"`
	Login      string    `json:"login"`
	AccountURL string    `json:"account_url"`
	Scope      string    `json:"scope"`
}

// LoginRequest is used for sending login credentials from clients.
type LoginRequest struct {
	Scope        string `json:"scope"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}
