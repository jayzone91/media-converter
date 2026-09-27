package web

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
)

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
		return nil, err
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
		return "localhost, Loopback-, Link-Local-, Metadata- und Adressen des Media-Converter-Servers können aus Sicherheitsgründen nicht gerendert werden. " +
			"Lokale Projekte müssen über eine zulässige Netzwerk-IP des Geräts erreichbar sein, auf dem das Projekt läuft."
	}

	return fmt.Sprintf(
		"localhost, Loopback-, Link-Local-, Metadata- und Adressen des Media-Converter-Servers können aus Sicherheitsgründen nicht gerendert werden. "+
			"Lokale Projekte müssen über eine zulässige Netzwerk-IP des Geräts erreichbar sein, auf dem das Projekt läuft. "+
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
		!ip.IsMulticast() &&
		!ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() &&
		!isCloudMetadataIP(ip)
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
