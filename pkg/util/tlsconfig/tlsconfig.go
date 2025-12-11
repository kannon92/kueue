/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tlsconfig

import (
	"crypto/tls"
	"fmt"
	"strings"

	config "sigs.k8s.io/kueue/apis/config/v1beta2"
)

var (
	// tlsVersionMap maps string TLS versions to crypto/tls constants
	tlsVersionMap = map[string]uint16{
		"1.2": tls.VersionTLS12,
		"1.3": tls.VersionTLS13,
	}

	// cipherSuiteMap maps cipher suite names to their crypto/tls constants
	cipherSuiteMap = map[string]uint16{
		// TLS 1.2 cipher suites
		"TLS_RSA_WITH_AES_128_CBC_SHA":                      tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		"TLS_RSA_WITH_AES_256_CBC_SHA":                      tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		"TLS_RSA_WITH_AES_128_GCM_SHA256":                   tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_RSA_WITH_AES_256_GCM_SHA384":                   tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":              tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":              tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":                tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":                tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256":           tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384":           tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":             tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":             tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256":       tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256":     tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		// TLS 1.3 cipher suites
		"TLS_AES_128_GCM_SHA256":       tls.TLS_AES_128_GCM_SHA256,
		"TLS_AES_256_GCM_SHA384":       tls.TLS_AES_256_GCM_SHA384,
		"TLS_CHACHA20_POLY1305_SHA256": tls.TLS_CHACHA20_POLY1305_SHA256,
	}

	// curvePreferencesMap maps curve names to their crypto/tls constants
	curvePreferencesMap = map[string]tls.CurveID{
		"CurveP256": tls.CurveP256,
		"CurveP384": tls.CurveP384,
		"CurveP521": tls.CurveP521,
		"X25519":    tls.X25519,
	}
)

// BuildTLSOptions converts TLSOptions from the configuration to controller-runtime TLSOpts
func BuildTLSOptions(cfg *config.TLSOptions) ([]func(*tls.Config), error) {
	if cfg == nil {
		return nil, nil
	}

	var tlsOpts []func(*tls.Config)

	tlsOpts = append(tlsOpts, func(c *tls.Config) {
		// Set minimum TLS version
		if cfg.MinTLSVersion != "" {
			version, ok := tlsVersionMap[cfg.MinTLSVersion]
			if !ok {
				// Default to TLS 1.2 if invalid version is provided
				version = tls.VersionTLS12
			}
			c.MinVersion = version
		} else {
			// Default to TLS 1.2
			c.MinVersion = tls.VersionTLS12
		}

		// Set cipher suites
		if len(cfg.CipherSuites) > 0 {
			cipherSuites, err := convertCipherSuites(cfg.CipherSuites)
			if err == nil && len(cipherSuites) > 0 {
				c.CipherSuites = cipherSuites
			}
		}

		// Set curve preferences
		if len(cfg.CurvePreferences) > 0 {
			curvePreferences, err := convertCurvePreferences(cfg.CurvePreferences)
			if err == nil && len(curvePreferences) > 0 {
				c.CurvePreferences = curvePreferences
			}
		}
	})

	return tlsOpts, nil
}

// convertCipherSuites converts cipher suite names to their crypto/tls constants
func convertCipherSuites(names []string) ([]uint16, error) {
	var suites []uint16
	var invalidSuites []string

	for _, name := range names {
		suite, ok := cipherSuiteMap[strings.TrimSpace(name)]
		if !ok {
			invalidSuites = append(invalidSuites, name)
			continue
		}
		suites = append(suites, suite)
	}

	if len(invalidSuites) > 0 {
		return suites, fmt.Errorf("invalid cipher suites: %v", invalidSuites)
	}

	return suites, nil
}

// convertCurvePreferences converts curve preference names to their crypto/tls constants
func convertCurvePreferences(names []string) ([]tls.CurveID, error) {
	var curves []tls.CurveID
	var invalidCurves []string

	for _, name := range names {
		curve, ok := curvePreferencesMap[strings.TrimSpace(name)]
		if !ok {
			invalidCurves = append(invalidCurves, name)
			continue
		}
		curves = append(curves, curve)
	}

	if len(invalidCurves) > 0 {
		return curves, fmt.Errorf("invalid curve preferences: %v", invalidCurves)
	}

	return curves, nil
}
