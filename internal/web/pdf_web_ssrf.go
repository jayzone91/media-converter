package web

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

const pdfWebDNSLookupTimeout = 5 * time.Second

var cloudMetadataIPs = []net.IP{
	net.ParseIP(
		"169.254.169.254",
	),
	net.ParseIP(
		"100.100.100.200",
	),
}

type pdfWebResolver interface {
	LookupIPAddr(
		context.Context,
		string,
	) ([]net.IPAddr, error)
}

func validatePDFWebTarget(
	ctx context.Context,
	target *url.URL,
) error {
	return validatePDFWebTargetWithResolver(
		ctx,
		target,
		net.DefaultResolver,
	)
}

func validatePDFWebRequest(
	ctx context.Context,
	rawURL string,
) error {
	target, err := validatePDFWebURL(
		rawURL,
	)
	if err != nil {
		return err
	}

	return validatePDFWebTarget(
		ctx,
		target,
	)
}

func validatePDFWebTargetWithResolver(
	ctx context.Context,
	target *url.URL,
	resolver pdfWebResolver,
) error {
	host :=
		strings.TrimSuffix(
			strings.ToLower(
				target.Hostname(),
			),
			".",
		)

	if host == "" {
		return fmt.Errorf(
			"target host is empty",
		)
	}

	if host == "localhost" {
		return fmt.Errorf(
			"localhost target",
		)
	}

	if ip := net.ParseIP(
		host,
	); ip != nil {
		return validatePDFWebIP(
			ip,
		)
	}

	lookupCtx, cancel :=
		context.WithTimeout(
			ctx,
			pdfWebDNSLookupTimeout,
		)
	defer cancel()

	addresses, err :=
		resolver.LookupIPAddr(
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
		if address.IP == nil {
			continue
		}

		if err := validatePDFWebIP(
			address.IP,
		); err != nil {
			return fmt.Errorf(
				"target host resolves to blocked address %s: %w",
				address.IP,
				err,
			)
		}
	}

	return nil
}

func validatePDFWebIP(
	ip net.IP,
) error {
	if ip == nil {
		return fmt.Errorf(
			"invalid IP address",
		)
	}

	switch {
	case ip.IsLoopback():
		return fmt.Errorf(
			"loopback address",
		)

	case ip.IsUnspecified():
		return fmt.Errorf(
			"unspecified address",
		)

	case ip.IsMulticast():
		return fmt.Errorf(
			"multicast address",
		)

	case ip.IsLinkLocalUnicast(),
		ip.IsLinkLocalMulticast():
		return fmt.Errorf(
			"link-local address",
		)

	case isCloudMetadataIP(ip):
		return fmt.Errorf(
			"cloud metadata address",
		)

	case isLocalServerIP(ip):
		return fmt.Errorf(
			"media-converter server address",
		)
	}

	return nil
}

func isCloudMetadataIP(
	ip net.IP,
) bool {
	for _, blocked := range cloudMetadataIPs {
		if blocked != nil &&
			ip.Equal(blocked) {
			return true
		}
	}

	return false
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

		if ip != nil &&
			target.Equal(ip) {
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
