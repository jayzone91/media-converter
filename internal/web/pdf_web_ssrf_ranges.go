package web

import (
	"fmt"
	"net"
	"os"
	"strings"
)

const pdfWebBlockedCIDRsEnv = "MEDIA_CONVERTER_WEB_PDF_BLOCKED_CIDRS"

func configuredPDFWebBlockedNetworks() (
	[]*net.IPNet,
	error,
) {
	value :=
		strings.TrimSpace(
			os.Getenv(
				pdfWebBlockedCIDRsEnv,
			),
		)

	if value == "" {
		return nil, nil
	}

	parts :=
		strings.Split(
			value,
			",",
		)

	networks :=
		make(
			[]*net.IPNet,
			0,
			len(parts),
		)

	for _, part := range parts {
		cidr :=
			strings.TrimSpace(
				part,
			)

		if cidr == "" {
			continue
		}

		_, network, err :=
			net.ParseCIDR(
				cidr,
			)

		if err != nil {
			return nil,
				fmt.Errorf(
					"invalid CIDR %q in %s: %w",
					cidr,
					pdfWebBlockedCIDRsEnv,
					err,
				)
		}

		networks = append(
			networks,
			network,
		)
	}

	return networks, nil
}

func validateConfiguredPDFWebNetwork(
	ip net.IP,
) error {
	networks, err :=
		configuredPDFWebBlockedNetworks()

	if err != nil {
		return err
	}

	for _, network := range networks {
		if network.Contains(
			ip,
		) {
			return fmt.Errorf(
				"address belongs to configured blocked network %s",
				network.String(),
			)
		}
	}

	return nil
}
