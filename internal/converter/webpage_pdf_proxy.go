package converter

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	webPDFProxyDNSLookupTimeout = 5 * time.Second
	webPDFProxyShutdownTimeout  = 2 * time.Second
)

type webPDFProxyResolver interface {
	LookupIPAddr(
		context.Context,
		string,
	) ([]net.IPAddr, error)
}

type webPDFProxy struct {
	listener net.Listener
	server   *http.Server

	validator WebPDFRequestValidator
	resolver  webPDFProxyResolver
}

func startWebPDFProxy(
	ctx context.Context,
	validator WebPDFRequestValidator,
) (*webPDFProxy, error) {
	if validator == nil {
		return nil, nil
	}

	listener, err :=
		net.Listen(
			"tcp",
			"127.0.0.1:0",
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"start webpage validation proxy: %w",
				err,
			)
	}

	proxy :=
		&webPDFProxy{
			listener: listener,

			validator: validator,

			resolver: net.DefaultResolver,
		}

	proxy.server =
		&http.Server{
			Handler: proxy,

			ReadHeaderTimeout: 10 * time.Second,

			IdleTimeout: 30 * time.Second,
		}

	go func() {
		_ = proxy.server.Serve(
			listener,
		)
	}()

	go func() {
		<-ctx.Done()

		_ = proxy.Close()
	}()

	return proxy, nil
}

func (p *webPDFProxy) URL() string {
	if p == nil ||
		p.listener == nil {
		return ""
	}

	return "http://" +
		p.listener.Addr().String()
}

func (p *webPDFProxy) Close() error {
	if p == nil ||
		p.server == nil {
		return nil
	}

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			webPDFProxyShutdownTimeout,
		)

	defer cancel()

	return p.server.Shutdown(
		ctx,
	)
}

func (p *webPDFProxy) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method ==
		http.MethodConnect {
		p.handleConnect(
			w,
			r,
		)

		return
	}

	p.handleHTTP(
		w,
		r,
	)
}

func (p *webPDFProxy) handleConnect(
	w http.ResponseWriter,
	r *http.Request,
) {
	target, err :=
		p.dialTarget(
			r.Context(),
			"https",
			"tcp",
			r.Host,
		)

	if err != nil {
		http.Error(
			w,
			"target blocked",
			http.StatusBadGateway,
		)

		return
	}

	hijacker, ok :=
		w.(http.Hijacker)

	if !ok {
		_ = target.Close()

		http.Error(
			w,
			"proxy hijacking unavailable",
			http.StatusInternalServerError,
		)

		return
	}

	client, buffered, err :=
		hijacker.Hijack()

	if err != nil {
		_ = target.Close()

		return
	}

	if _, err :=
		buffered.WriteString(
			"HTTP/1.1 200 Connection Established\r\n\r\n",
		); err != nil {
		_ = client.Close()
		_ = target.Close()

		return
	}

	if err :=
		buffered.Flush(); err != nil {
		_ = client.Close()
		_ = target.Close()

		return
	}

	done :=
		make(
			chan struct{},
			2,
		)

	go func() {
		_, _ = io.Copy(
			target,
			buffered.Reader,
		)

		done <- struct{}{}
	}()

	go func() {
		_, _ = io.Copy(
			client,
			target,
		)

		done <- struct{}{}
	}()

	<-done

	_ = client.Close()
	_ = target.Close()
}

func (p *webPDFProxy) handleHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.URL == nil ||
		r.URL.Scheme == "" ||
		r.URL.Host == "" {
		http.Error(
			w,
			"invalid proxy request",
			http.StatusBadRequest,
		)

		return
	}

	request :=
		r.Clone(
			r.Context(),
		)

	request.RequestURI =
		""

	removeWebPDFProxyHopHeaders(
		request.Header,
	)

	scheme :=
		strings.ToLower(
			request.URL.Scheme,
		)

	transport :=
		&http.Transport{
			Proxy: nil,

			DisableKeepAlives: true,

			DialContext: func(
				ctx context.Context,
				network string,
				address string,
			) (net.Conn, error) {
				return p.dialTarget(
					ctx,
					scheme,
					network,
					address,
				)
			},
		}

	defer transport.CloseIdleConnections()

	response, err :=
		transport.RoundTrip(
			request,
		)

	if err != nil {
		http.Error(
			w,
			"target unavailable",
			http.StatusBadGateway,
		)

		return
	}

	defer response.Body.Close()

	removeWebPDFProxyHopHeaders(
		response.Header,
	)

	for name, values := range response.Header {
		for _, value := range values {
			w.Header().Add(
				name,
				value,
			)
		}
	}

	w.WriteHeader(
		response.StatusCode,
	)

	_, _ = io.Copy(
		w,
		response.Body,
	)
}
