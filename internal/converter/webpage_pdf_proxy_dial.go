package converter

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

func (p *webPDFProxy) dialTarget(
	ctx context.Context,
	scheme string,
	network string,
	address string,
) (net.Conn, error) {
	host, port, err :=
		webPDFProxyHostPort(
			scheme,
			address,
		)

	if err != nil {
		return nil, err
	}

	host =
		strings.TrimSuffix(
			strings.ToLower(
				host,
			),
			".",
		)

	if host == "" {
		return nil,
			fmt.Errorf(
				"proxy target host is empty",
			)
	}

	if ip :=
		net.ParseIP(
			host,
		); ip != nil {
		if err :=
			p.validateResolvedIP(
				ctx,
				scheme,
				ip,
				port,
			); err != nil {
			return nil, err
		}

		return dialWebPDFResolvedIP(
			ctx,
			network,
			ip,
			port,
		)
	}

	resolved, err :=
		resolveValidatedWebPDFIPs(
			ctx,
			p,
			scheme,
			host,
			port,
		)

	if err != nil {
		return nil, err
	}

	var lastErr error

	for _, ip := range resolved {
		connection, err :=
			dialWebPDFResolvedIP(
				ctx,
				network,
				ip,
				port,
			)

		if err == nil {
			return connection,
				nil
		}

		lastErr =
			err
	}

	return nil,
		fmt.Errorf(
			"connect proxy target %q: %w",
			host,
			lastErr,
		)
}

func resolveValidatedWebPDFIPs(
	ctx context.Context,
	proxy *webPDFProxy,
	scheme string,
	host string,
	port string,
) ([]net.IP, error) {
	if proxy == nil {
		return nil,
			fmt.Errorf(
				"web PDF proxy is nil",
			)
	}

	if proxy.resolver == nil {
		return nil,
			fmt.Errorf(
				"web PDF proxy resolver is nil",
			)
	}

	lookupCtx, cancel :=
		context.WithTimeout(
			ctx,
			webPDFProxyDNSLookupTimeout,
		)

	defer cancel()

	addresses, err :=
		proxy.resolver.LookupIPAddr(
			lookupCtx,
			host,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"resolve proxy target %q: %w",
				host,
				err,
			)
	}

	if len(addresses) == 0 {
		return nil,
			fmt.Errorf(
				"proxy target %q has no addresses",
				host,
			)
	}

	resolved :=
		make(
			[]net.IP,
			0,
			len(addresses),
		)

	for _, address := range addresses {
		if address.IP == nil {
			continue
		}

		if err :=
			proxy.validateResolvedIP(
				ctx,
				scheme,
				address.IP,
				port,
			); err != nil {
			return nil,
				fmt.Errorf(
					"proxy target %q resolved to blocked address %s: %w",
					host,
					address.IP,
					err,
				)
		}

		resolved = append(
			resolved,
			address.IP,
		)
	}

	if len(resolved) == 0 {
		return nil,
			fmt.Errorf(
				"proxy target %q has no usable addresses",
				host,
			)
	}

	return resolved, nil
}

func webPDFProxyHostPort(
	scheme string,
	address string,
) (string, string, error) {
	host, port, err :=
		net.SplitHostPort(
			address,
		)

	if err == nil {
		return host,
			port,
			nil
	}

	if strings.Contains(
		address,
		":",
	) {
		if ip :=
			net.ParseIP(
				address,
			); ip != nil {
			return ip.String(),
				webPDFDefaultPort(
					scheme,
				),
				nil
		}
	}

	if address == "" {
		return "", "",
			fmt.Errorf(
				"proxy target address is empty",
			)
	}

	return address,
		webPDFDefaultPort(
			scheme,
		),
		nil
}

func webPDFDefaultPort(
	scheme string,
) string {
	if strings.EqualFold(
		scheme,
		"https",
	) {
		return "443"
	}

	return "80"
}

func (p *webPDFProxy) validateResolvedIP(
	ctx context.Context,
	scheme string,
	ip net.IP,
	port string,
) error {
	if p == nil ||
		p.validator == nil {
		return fmt.Errorf(
			"web PDF proxy validator is unavailable",
		)
	}

	target :=
		&url.URL{
			Scheme: scheme,

			Host: net.JoinHostPort(
				ip.String(),
				port,
			),
		}

	if err :=
		p.validator(
			ctx,
			target.String(),
		); err != nil {
		return fmt.Errorf(
			"resolved target rejected: %w",
			err,
		)
	}

	return nil
}

func dialWebPDFResolvedIP(
	ctx context.Context,
	network string,
	ip net.IP,
	port string,
) (net.Conn, error) {
	dialer :=
		&net.Dialer{
			Timeout: 15 * time.Second,

			KeepAlive: 15 * time.Second,
		}

	return dialer.DialContext(
		ctx,
		network,
		net.JoinHostPort(
			ip.String(),
			port,
		),
	)
}

func removeWebPDFProxyHopHeaders(
	header map[string][]string,
) {
	for _, name := range []string{
		"Connection",
		"Proxy-Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	} {
		delete(
			header,
			name,
		)
	}
}
