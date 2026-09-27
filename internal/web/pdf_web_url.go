package web

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const pdfWebDNSLookupTimeout = 5 * time.Second

func validatePDFWebURL(
	rawURL string,
) (*url.URL, error) {
	if rawURL == "" {
		return nil,
			fmt.Errorf(
				"URL is empty",
			)
	}

	parsedURL, err :=
		url.Parse(
			rawURL,
		)

	if err != nil {
		return nil,
			err
	}

	scheme :=
		strings.ToLower(
			parsedURL.Scheme,
		)

	if scheme != "http" &&
		scheme != "https" {
		return nil,
			fmt.Errorf(
				"unsupported URL scheme",
			)
	}

	if parsedURL.Host == "" {
		return nil,
			fmt.Errorf(
				"URL has no host",
			)
	}

	if parsedURL.User != nil {
		return nil,
			fmt.Errorf(
				"URL credentials are not allowed",
			)
	}

	return parsedURL,
		nil
}

func validatePDFWebTarget(
	ctx context.Context,
	parsedURL *url.URL,
) error {
	host :=
		strings.TrimSuffix(
			strings.ToLower(
				parsedURL.Hostname(),
			),
			".",
		)

	if host == "localhost" {
		return fmt.Errorf(
			"loopback target",
		)
	}

	if ip :=
		net.ParseIP(
			host,
		); ip != nil {
		if ip.IsLoopback() {
			return fmt.Errorf(
				"loopback target",
			)
		}

		if isLocalServerIP(
			ip,
		) {
			return fmt.Errorf(
				"server target",
			)
		}

		return nil
	}

	lookupCtx, cancel :=
		context.WithTimeout(
			ctx,
			pdfWebDNSLookupTimeout,
		)
	defer cancel()

	addresses, err :=
		net.DefaultResolver.LookupIPAddr(
			lookupCtx,
			host,
		)

	if err != nil {
		return fmt.Errorf(
			"resolve target host: %w",
			err,
		)
	}

	if len(addresses) == 0 {
		return fmt.Errorf(
			"target host has no addresses",
		)
	}

	for _, address := range addresses {
		ip :=
			address.IP

		if ip == nil {
			continue
		}

		if ip.IsLoopback() {
			return fmt.Errorf(
				"loopback target",
			)
		}

		if isLocalServerIP(
			ip,
		) {
			return fmt.Errorf(
				"server target",
			)
		}
	}

	return nil
}

func isLocalServerIP(
	target net.IP,
) bool {
	if target == nil {
		return false
	}

	addresses, err :=
		net.InterfaceAddrs()

	if err != nil {
		return false
	}

	for _, address := range addresses {
		ip :=
			interfaceAddressIP(
				address,
			)

		if ip == nil {
			continue
		}

		if target.Equal(
			ip,
		) {
			return true
		}
	}

	return false
}

func interfaceAddressIP(
	address net.Addr,
) net.IP {
	switch value :=
		address.(type) {
	case *net.IPNet:
		return value.IP

	case *net.IPAddr:
		return value.IP

	default:
		host :=
			address.String()

		if slash :=
			strings.IndexByte(
				host,
				'/',
			); slash >= 0 {
			host =
				host[:slash]
		}

		return net.ParseIP(
			host,
		)
	}
}

func pdfWebTargetError(
	r *http.Request,
	parsedURL *url.URL,
) string {
	suggestion :=
		pdfWebSuggestedDeviceURL(
			r,
			parsedURL,
		)

	if suggestion == "" {
		return "localhost, Loopback-Adressen und Adressen des Media-Converter-Servers können aus Sicherheitsgründen nicht gerendert werden. " +
			"Lokale Projekte müssen über die Netzwerk-IP des Geräts erreichbar sein, auf dem das Projekt läuft."
	}

	return fmt.Sprintf(
		"localhost, Loopback-Adressen und Adressen des Media-Converter-Servers können aus Sicherheitsgründen nicht gerendert werden. "+
			"Lokale Projekte müssen über die Netzwerk-IP des Geräts erreichbar sein, auf dem das Projekt läuft. "+
			"Versuche stattdessen: %s",
		suggestion,
	)
}

func pdfWebSuggestedDeviceURL(
	r *http.Request,
	target *url.URL,
) string {
	clientIP :=
		pdfWebClientIP(
			r,
		)

	if clientIP == nil {
		return ""
	}

	suggestion :=
		*target

	port :=
		target.Port()

	if port != "" {
		suggestion.Host =
			net.JoinHostPort(
				clientIP.String(),
				port,
			)
	} else {
		suggestion.Host =
			formatPDFWebURLHost(
				clientIP,
			)
	}

	suggestion.User =
		nil

	return suggestion.String()
}

func pdfWebClientIP(
	r *http.Request,
) net.IP {
	candidates :=
		pdfWebClientIPCandidates(
			r,
		)

	for _, candidate := range candidates {
		ip :=
			net.ParseIP(
				candidate,
			)

		if isUsablePDFWebClientIP(
			ip,
		) &&
			ip.IsPrivate() &&
			!isLocalServerIP(
				ip,
			) {
			return ip
		}
	}

	for _, candidate := range candidates {
		ip :=
			net.ParseIP(
				candidate,
			)

		if isUsablePDFWebClientIP(
			ip,
		) &&
			!isLocalServerIP(
				ip,
			) {
			return ip
		}
	}

	return nil
}

func pdfWebClientIPCandidates(
	r *http.Request,
) []string {
	var candidates []string

	for _, value := range strings.Split(
		r.Header.Get(
			"X-Forwarded-For",
		),
		",",
	) {
		value =
			strings.TrimSpace(
				value,
			)

		if value != "" {
			candidates =
				append(
					candidates,
					value,
				)
		}
	}

	if realIP :=
		strings.TrimSpace(
			r.Header.Get(
				"X-Real-IP",
			),
		); realIP != "" {
		candidates =
			append(
				candidates,
				realIP,
			)
	}

	remoteHost, _, err :=
		net.SplitHostPort(
			r.RemoteAddr,
		)

	if err == nil &&
		remoteHost != "" {
		candidates =
			append(
				candidates,
				remoteHost,
			)
	} else if r.RemoteAddr != "" {
		candidates =
			append(
				candidates,
				r.RemoteAddr,
			)
	}

	return candidates
}

func isUsablePDFWebClientIP(
	ip net.IP,
) bool {
	if ip == nil {
		return false
	}

	return !ip.IsLoopback() &&
		!ip.IsUnspecified() &&
		!ip.IsMulticast()
}

func formatPDFWebURLHost(
	ip net.IP,
) string {
	if ip.To4() != nil {
		return ip.String()
	}

	return "[" +
		ip.String() +
		"]"
}
