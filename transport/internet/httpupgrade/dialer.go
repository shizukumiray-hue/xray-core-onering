package httpupgrade

import (
	"bufio"
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/onering"
	"github.com/xtls/xray-core/common/utils"
	"github.com/xtls/xray-core/transport/internet"
	"github.com/xtls/xray-core/transport/internet/stat"
	"github.com/xtls/xray-core/transport/internet/tls"
)

type ConnRF struct {
	net.Conn
	Req   *http.Request
	First bool
}

func (c *ConnRF) Read(b []byte) (int, error) {
	if c.First {
		c.First = false
		// create reader capped to size of `b`, so it can be fully drained into
		// `b` later with a single Read call
		reader := bufio.NewReaderSize(c.Conn, len(b))
		resp, err := http.ReadResponse(reader, c.Req) // nolint:bodyclose
		if err != nil {
			return 0, err
		}
		if resp.Status != "101 Switching Protocols" ||
			strings.ToLower(resp.Header.Get("Upgrade")) != "websocket" ||
			strings.ToLower(resp.Header.Get("Connection")) != "upgrade" {
			return 0, errors.New("unrecognized reply")
		}
		// drain remaining bufreader
		return reader.Read(b[:reader.Buffered()])
	}
	return c.Conn.Read(b)
}

func dialhttpUpgrade(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (net.Conn, error) {
	tConfig := tls.ConfigFromStreamSettings(streamSettings)

	// Get Onering config from TLS settings if available
	var oneringCfg *onering.Config
	if tConfig != nil {
		oneringCfg = tConfig.GetOneringConfig()
	}

	// Apply timing jitter for DPI evasion (Phase 2)
	if oneringCfg != nil && oneringCfg.Enabled && oneringCfg.MultiCDNEnabled && oneringCfg.MultiCDNManager != nil {
		if err := oneringCfg.MultiCDNManager.ApplyJitter(ctx); err != nil {
			// Context cancelled during jitter, propagate error
			return nil, err
		}
	}

	// Multi-CDN retry logic
	if oneringCfg != nil && oneringCfg.Enabled && oneringCfg.MultiCDNEnabled && oneringCfg.MultiCDNManager != nil {
		return dialhttpUpgradeWithMultiCDN(ctx, dest, streamSettings, oneringCfg)
	}

	// Single-CDN or no onering
	return dialhttpUpgradeSingle(ctx, dest, streamSettings, oneringCfg)
}

// dialhttpUpgradeSingle performs a single HTTP upgrade connection
func dialhttpUpgradeSingle(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig, oneringCfg *onering.Config) (net.Conn, error) {
	transportConfiguration := streamSettings.ProtocolSettings.(*Config)
	tConfig := tls.ConfigFromStreamSettings(streamSettings)

	// Override destination if Onering is enabled
	actualDest := dest
	if oneringCfg != nil && oneringCfg.Enabled {
		bugDomain := oneringCfg.GetDialAddress()
		actualDest = net.Destination{
			Network: dest.Network,
			Address: net.ParseAddress(bugDomain),
			Port:    dest.Port,
		}
	}

	pconn, err := internet.DialSystem(ctx, actualDest, streamSettings.SocketSettings)
	if err != nil {
		errors.LogErrorInner(ctx, err, "failed to dial to ", dest)
		return nil, err
	}

	if streamSettings.TcpmaskManager != nil {
		newConn, err := streamSettings.TcpmaskManager.WrapConnClient(pconn)
		if err != nil {
			pconn.Close()
			return nil, errors.New("mask err").Base(err)
		}
		pconn = newConn
	}

	var conn net.Conn
	var requestURL url.URL
	if tConfig != nil {
		tlsConfig := tConfig.GetTLSConfig(tls.WithDestination(dest), tls.WithNextProto("http/1.1"))
		// Override ServerName with bug domain if Onering is enabled
		if oneringCfg != nil && oneringCfg.Enabled {
			tlsConfig.ServerName = oneringCfg.GetTLSSNI()
		}
		if fingerprint := tls.GetFingerprint(tConfig.Fingerprint); fingerprint != nil {
			conn = tls.UClient(pconn, tlsConfig, fingerprint)
			if err := conn.(*tls.UConn).WebsocketHandshakeContext(ctx); err != nil {
				return nil, err
			}
		} else {
			conn = tls.Client(pconn, tlsConfig)
		}
		requestURL.Scheme = "https"
	} else {
		conn = pconn
		requestURL.Scheme = "http"
	}

	requestURL.Host = transportConfiguration.Host
	if requestURL.Host == "" && oneringCfg != nil && oneringCfg.Enabled {
		requestURL.Host = oneringCfg.RealDomain
	} else if requestURL.Host == "" && tConfig != nil {
		requestURL.Host = tConfig.ServerName
	}
	if requestURL.Host == "" {
		requestURL.Host = dest.Address.String()
	}
	requestURL.Path = transportConfiguration.GetNormalizedPath()
	req := &http.Request{
		Method: http.MethodGet,
		URL:    &requestURL,
		Header: make(http.Header),
	}
	for key, value := range transportConfiguration.Header {
		AddHeader(req.Header, key, value)
	}
	utils.TryDefaultHeadersWith(req.Header, "ws")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")

	err = req.Write(conn)
	if err != nil {
		return nil, err
	}

	connRF := &ConnRF{
		Conn:  conn,
		Req:   req,
		First: true,
	}

	if transportConfiguration.Ed == 0 {
		_, err = connRF.Read([]byte{})
		if err != nil {
			return nil, err
		}
	}

	return connRF, nil
}

// dialhttpUpgradeWithMultiCDN attempts HTTP upgrade with multi-CDN failover
func dialhttpUpgradeWithMultiCDN(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig, oneringCfg *onering.Config) (net.Conn, error) {
	maxRetries := 3
	if oneringCfg.MultiCDNManager != nil {
		providers := oneringCfg.MultiCDNManager.GetProviders()
		if len(providers) > 0 {
			maxRetries = len(providers) * 2
		}
	}

	var lastErr error
	triedProviders := make(map[string]bool)

	for attempt := 0; attempt < maxRetries; attempt++ {
		provider := oneringCfg.MultiCDNManager.SelectCDN()
		if provider == nil {
			break
		}

		if triedProviders[provider.Name] {
			continue
		}
		triedProviders[provider.Name] = true

		// Create modified config with this provider's bug domain
		modifiedCfg := &onering.Config{
			Enabled:         true,
			RealDomain:      oneringCfg.RealDomain,
			BugDomain:       provider.BugDomain,
			MultiCDNEnabled: false, // Treat as single-CDN for this attempt
		}

		conn, err := dialhttpUpgradeSingle(ctx, dest, streamSettings, modifiedCfg)
		if err == nil {
			// Success - use thread-safe method - fixes BUG #5
			if oneringCfg.MultiCDNManager != nil {
				oneringCfg.MultiCDNManager.RecordSuccess(provider.Name, 0)
			}
			return conn, nil
		}

		// Failed - use thread-safe method - fixes BUG #5
		if oneringCfg.MultiCDNManager != nil {
			oneringCfg.MultiCDNManager.RecordFailure(provider.Name)
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, errors.New("failed to dial HTTP upgrade with multi-CDN after all retries").Base(lastErr)
	}
	return nil, errors.New("no available CDN providers")
}

// http.Header.Add() will convert headers to MIME header format.
// Some people don't like this because they want to send "Web*S*ocket".
// So we add a simple function to replace that method.
func AddHeader(header http.Header, key, value string) {
	header[key] = append(header[key], value)
}

func Dial(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (stat.Connection, error) {
	errors.LogInfo(ctx, "creating connection to ", dest)

	conn, err := dialhttpUpgrade(ctx, dest, streamSettings)
	if err != nil {
		return nil, errors.New("failed to dial request to ", dest).Base(err)
	}
	return stat.Connection(conn), nil
}

func init() {
	common.Must(internet.RegisterTransportDialer(protocolName, Dial))
}
