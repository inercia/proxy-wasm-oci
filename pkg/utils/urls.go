package utils

import "net/url"

// ExtractHostname returns hostname from URL
func ExtractHostname(addr string) (string, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return "", err
	}
	return u.Hostname(), nil
}
