package netutil

import (
	"fmt"
	"net"
	"net/url"
	"syscall"
)

// PreflightDNS validates that the URL is well-formed and its host resolves.
func PreflightDNS(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}

	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("missing host in url")
	}

	if _, err := net.LookupHost(host); err != nil {
		return fmt.Errorf("dns resolution failed for host %q: %w", host, err)
	}
	return nil
}

// CheckUlimitWarning inspects the soft RLIMIT_NOFILE and returns a warning
// if the requested number of connections appears to exceed it.
// On non-Unix platforms this becomes a no-op.
func CheckUlimitWarning(requestedConns int) error {
	if requestedConns <= 0 {
		return nil
	}

	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		// Best effort only; if this fails we stay silent.
		return nil
	}

	if uint64(requestedConns) > rLimit.Cur {
		return fmt.Errorf("requested connections (%d) exceed soft open-files limit (%d)", requestedConns, rLimit.Cur)
	}

	return nil
}


