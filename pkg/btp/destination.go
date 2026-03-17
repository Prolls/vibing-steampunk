// Package btp provides SAP BTP platform services integration.
package btp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// DestinationConfig holds the SAP connection details resolved from a BTP Destination.
type DestinationConfig struct {
	URL      string
	Username string
	Password string
}

// vcapDestinationBinding represents a single destination service binding in VCAP_SERVICES.
type vcapDestinationBinding struct {
	Credentials struct {
		ClientID        string `json:"clientid"`
		ClientSecret    string `json:"clientsecret"`
		TokenServiceURL string `json:"tokenServiceURL"`
		URI             string `json:"uri"`
		URL             string `json:"url"` // fallback if URI is absent
	} `json:"credentials"`
	Name string `json:"name"`
}

// ResolveDestination reads the BTP Destination service from VCAP_SERVICES,
// fetches an OAuth token, and retrieves the named destination's connection details.
//
// The destination must be of type HTTP with BasicAuthentication.
// The CF app must be bound to a destination service instance.
func ResolveDestination(destinationName string) (*DestinationConfig, error) {
	raw := os.Getenv("VCAP_SERVICES")
	if raw == "" {
		return nil, fmt.Errorf("VCAP_SERVICES not set — is vsp bound to a destination service instance?")
	}

	var services map[string][]vcapDestinationBinding
	if err := json.Unmarshal([]byte(raw), &services); err != nil {
		return nil, fmt.Errorf("failed to parse VCAP_SERVICES: %w", err)
	}

	// BTP destination service appears under the "destination" key
	bindings, ok := services["destination"]
	if !ok || len(bindings) == 0 {
		return nil, fmt.Errorf("no destination service binding found in VCAP_SERVICES (bind vsp to a destination service instance)")
	}
	creds := bindings[0].Credentials

	// Resolve the service endpoint URI (try URI first, fall back to URL)
	serviceURI := creds.URI
	if serviceURI == "" {
		serviceURI = creds.URL
	}
	if serviceURI == "" {
		return nil, fmt.Errorf("destination service binding has no URI/URL field")
	}

	// Fetch OAuth token using client_credentials grant
	// tokenServiceURL may be absent; fall back to the service URL (XSUAA base) + /oauth/token
	tokenServiceURL := creds.TokenServiceURL
	if tokenServiceURL == "" {
		tokenServiceURL = creds.URL
	}
	if tokenServiceURL == "" {
		tokenServiceURL = serviceURI
	}
	token, err := fetchClientCredentialsToken(tokenServiceURL, creds.ClientID, creds.ClientSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain destination service token: %w", err)
	}

	// Call Destination service API
	endpoint := strings.TrimRight(serviceURI, "/") +
		"/destination-configuration/v1/destinations/" + url.PathEscape(destinationName)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build destination request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("destination service request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("destination service returned HTTP %d for destination %q: %s",
			resp.StatusCode, destinationName, string(body))
	}

	// Parse the destination configuration
	var result struct {
		DestinationConfiguration struct {
			URL            string `json:"URL"`
			Authentication string `json:"Authentication"`
			User           string `json:"User"`
			Password       string `json:"Password"`
		} `json:"destinationConfiguration"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse destination response: %w", err)
	}

	dc := result.DestinationConfiguration
	if dc.URL == "" {
		return nil, fmt.Errorf("destination %q has no URL field", destinationName)
	}
	if dc.Authentication != "BasicAuthentication" && dc.Authentication != "" {
		return nil, fmt.Errorf("destination %q uses unsupported authentication type %q (expected BasicAuthentication)",
			destinationName, dc.Authentication)
	}
	if dc.User == "" || dc.Password == "" {
		return nil, fmt.Errorf("destination %q is missing User or Password fields", destinationName)
	}

	return &DestinationConfig{
		URL:      dc.URL,
		Username: dc.User,
		Password: dc.Password,
	}, nil
}

// fetchClientCredentialsToken obtains an OAuth2 access token using client_credentials grant.
func fetchClientCredentialsToken(tokenServiceURL, clientID, clientSecret string) (string, error) {
	// tokenServiceURL may already include /oauth/token or may be just the base URL
	tokenEndpoint := tokenServiceURL
	if !strings.Contains(tokenEndpoint, "/oauth/token") {
		tokenEndpoint = strings.TrimRight(tokenEndpoint, "/") + "/oauth/token"
	}

	params := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.PostForm(tokenEndpoint, params)
	if err != nil {
		return "", fmt.Errorf("token request to %s failed: %w", tokenEndpoint, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}
	if tokenResp.Error != "" {
		return "", fmt.Errorf("token error %q: %s", tokenResp.Error, tokenResp.Description)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("token response contains no access_token")
	}

	return tokenResp.AccessToken, nil
}
