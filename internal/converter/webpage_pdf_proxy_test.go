package converter

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
)

type sequenceWebPDFResolver struct {
	mu sync.Mutex

	responses [][]net.IPAddr
	index     int
}

func (r *sequenceWebPDFResolver) LookupIPAddr(
	_ context.Context,
	_ string,
) ([]net.IPAddr, error) {
	r.mu.Lock()

	defer r.mu.Unlock()

	if len(r.responses) == 0 {
		return nil,
			fmt.Errorf(
				"no DNS responses configured",
			)
	}

	index :=
		r.index

	if index >=
		len(r.responses) {
		index =
			len(r.responses) -
				1
	}

	r.index++

	return r.responses[index],
		nil
}

func TestWebPDFProxyHostPortDefaultsHTTP(
	t *testing.T,
) {
	t.Parallel()

	host, port, err :=
		webPDFProxyHostPort(
			"http",
			"example.com",
		)

	if err != nil {
		t.Fatalf(
			"webPDFProxyHostPort returned error: %v",
			err,
		)
	}

	if host !=
		"example.com" {
		t.Fatalf(
			"unexpected host: %q",
			host,
		)
	}

	if port !=
		"80" {
		t.Fatalf(
			"unexpected port: %q",
			port,
		)
	}
}

func TestWebPDFProxyHostPortDefaultsHTTPS(
	t *testing.T,
) {
	t.Parallel()

	host, port, err :=
		webPDFProxyHostPort(
			"https",
			"example.com",
		)

	if err != nil {
		t.Fatalf(
			"webPDFProxyHostPort returned error: %v",
			err,
		)
	}

	if host !=
		"example.com" {
		t.Fatalf(
			"unexpected host: %q",
			host,
		)
	}

	if port !=
		"443" {
		t.Fatalf(
			"unexpected port: %q",
			port,
		)
	}
}

func TestWebPDFProxyRejectsResolvedLoopback(
	t *testing.T,
) {
	t.Parallel()

	proxy :=
		&webPDFProxy{
			resolver: &sequenceWebPDFResolver{
				responses: [][]net.IPAddr{
					{
						{
							IP: net.ParseIP(
								"127.42.23.7",
							),
						},
					},
				},
			},

			validator: rejectPrivateWebPDFTarget,
		}

	_, err :=
		proxy.dialTarget(
			context.Background(),
			"http",
			"tcp",
			"example.test:80",
		)

	if err == nil {
		t.Fatal(
			"expected loopback target to be rejected",
		)
	}

	if !strings.Contains(
		err.Error(),
		"blocked",
	) {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestWebPDFProxyRejectsMixedPublicAndBlockedResolution(
	t *testing.T,
) {
	t.Parallel()

	proxy :=
		&webPDFProxy{
			resolver: &sequenceWebPDFResolver{
				responses: [][]net.IPAddr{
					{
						{
							IP: net.ParseIP(
								"93.184.216.34",
							),
						},
						{
							IP: net.ParseIP(
								"127.0.0.1",
							),
						},
					},
				},
			},

			validator: rejectPrivateWebPDFTarget,
		}

	_, err :=
		proxy.dialTarget(
			context.Background(),
			"http",
			"tcp",
			"example.test:80",
		)

	if err == nil {
		t.Fatal(
			"expected mixed DNS result to be rejected",
		)
	}
}

func TestWebPDFProxyRejectsDNSRebinding(
	t *testing.T,
) {
	t.Parallel()

	resolver :=
		&sequenceWebPDFResolver{
			responses: [][]net.IPAddr{
				{
					{
						IP: net.ParseIP(
							"93.184.216.34",
						),
					},
				},

				{
					{
						IP: net.ParseIP(
							"127.0.0.1",
						),
					},
				},
			},
		}

	proxy :=
		&webPDFProxy{
			resolver: resolver,

			validator: rejectPrivateWebPDFTarget,
		}

	/*
		Erste DNS-Auflösung simuliert den zunächst
		öffentlichen Host.

		Wir testen hier nur die Auflösungs-/Validierungslogik,
		nicht den tatsächlichen Netzwerk-Dial.
	*/
	first, err :=
		resolveValidatedWebPDFIPs(
			context.Background(),
			proxy,
			"http",
			"example.test",
			"80",
		)

	if err != nil {
		t.Fatalf(
			"first DNS response should be accepted: %v",
			err,
		)
	}

	if len(first) !=
		1 {
		t.Fatalf(
			"expected one public address, got %d",
			len(first),
		)
	}

	/*
		Beim nächsten Verbindungsversuch liefert derselbe
		Hostname Loopback. Das muss vor dem Dial scheitern.
	*/
	_, err =
		resolveValidatedWebPDFIPs(
			context.Background(),
			proxy,
			"http",
			"example.test",
			"80",
		)

	if err == nil {
		t.Fatal(
			"expected rebinding to loopback to be rejected",
		)
	}
}

func rejectPrivateWebPDFTarget(
	_ context.Context,
	rawURL string,
) error {
	host, _, err :=
		net.SplitHostPort(
			mustWebPDFURLHost(
				rawURL,
			),
		)

	if err != nil {
		return err
	}

	ip :=
		net.ParseIP(
			host,
		)

	if ip == nil {
		return fmt.Errorf(
			"invalid target IP",
		)
	}

	if ip.IsLoopback() {
		return fmt.Errorf(
			"blocked loopback address",
		)
	}

	return nil
}

func mustWebPDFURLHost(
	rawURL string,
) string {
	const separator = "://"

	index :=
		strings.Index(
			rawURL,
			separator,
		)

	if index < 0 {
		return rawURL
	}

	return rawURL[index+
		len(separator):]
}
