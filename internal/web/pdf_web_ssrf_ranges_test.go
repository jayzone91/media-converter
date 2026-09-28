package web

import (
	"net"
	"testing"
)

func TestConfiguredPDFWebBlockedNetworksEmpty(
	t *testing.T,
) {
	t.Setenv(
		pdfWebBlockedCIDRsEnv,
		"",
	)

	networks, err :=
		configuredPDFWebBlockedNetworks()

	if err != nil {
		t.Fatalf(
			"configuredPDFWebBlockedNetworks returned error: %v",
			err,
		)
	}

	if len(networks) != 0 {
		t.Fatalf(
			"expected no configured networks, got %d",
			len(networks),
		)
	}
}

func TestConfiguredPDFWebBlockedNetworks(
	t *testing.T,
) {
	t.Setenv(
		pdfWebBlockedCIDRsEnv,
		"172.17.0.0/16, 10.50.0.0/24, fd00:1234::/48",
	)

	networks, err :=
		configuredPDFWebBlockedNetworks()

	if err != nil {
		t.Fatalf(
			"configuredPDFWebBlockedNetworks returned error: %v",
			err,
		)
	}

	if len(networks) != 3 {
		t.Fatalf(
			"expected 3 configured networks, got %d",
			len(networks),
		)
	}
}

func TestConfiguredPDFWebBlockedNetworksRejectInvalidCIDR(
	t *testing.T,
) {
	t.Setenv(
		pdfWebBlockedCIDRsEnv,
		"172.17.0.0/16,not-a-network",
	)

	_, err :=
		configuredPDFWebBlockedNetworks()

	if err == nil {
		t.Fatal(
			"expected invalid CIDR to fail",
		)
	}
}

func TestValidateConfiguredPDFWebNetworkBlocksIPv4(
	t *testing.T,
) {
	t.Setenv(
		pdfWebBlockedCIDRsEnv,
		"172.17.0.0/16",
	)

	err :=
		validateConfiguredPDFWebNetwork(
			net.ParseIP(
				"172.17.23.42",
			),
		)

	if err == nil {
		t.Fatal(
			"expected configured IPv4 network to be blocked",
		)
	}
}

func TestValidateConfiguredPDFWebNetworkAllowsOtherIPv4(
	t *testing.T,
) {
	t.Setenv(
		pdfWebBlockedCIDRsEnv,
		"172.17.0.0/16",
	)

	err :=
		validateConfiguredPDFWebNetwork(
			net.ParseIP(
				"10.20.30.40",
			),
		)

	if err != nil {
		t.Fatalf(
			"expected unrelated IPv4 address to be allowed: %v",
			err,
		)
	}
}

func TestValidateConfiguredPDFWebNetworkBlocksIPv6(
	t *testing.T,
) {
	t.Setenv(
		pdfWebBlockedCIDRsEnv,
		"fd00:1234::/48",
	)

	err :=
		validateConfiguredPDFWebNetwork(
			net.ParseIP(
				"fd00:1234::10",
			),
		)

	if err == nil {
		t.Fatal(
			"expected configured IPv6 network to be blocked",
		)
	}
}
