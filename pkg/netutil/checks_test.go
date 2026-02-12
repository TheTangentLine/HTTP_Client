package netutil

import (
	"strings"
	"testing"
)

func TestPreflightDNS_InvalidURL(t *testing.T) {
	err := PreflightDNS("://no-scheme")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	if !strings.Contains(err.Error(), "invalid url") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPreflightDNS_MissingHost(t *testing.T) {
	err := PreflightDNS("http://")
	if err == nil {
		t.Fatal("expected error for missing host")
	}
	if !strings.Contains(err.Error(), "missing host") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPreflightDNS_ValidResolvableHost(t *testing.T) {
	// 127.0.0.1 and localhost typically resolve
	for _, url := range []string{"http://127.0.0.1/", "http://localhost/"} {
		err := PreflightDNS(url)
		if err != nil {
			t.Errorf("PreflightDNS(%q): %v", url, err)
		}
	}
}

func TestPreflightDNS_UnresolvableHost(t *testing.T) {
	// Use a TLD that is reserved for "no such host" by RFC 6761
	err := PreflightDNS("http://nonexistent.invalid/")
	if err == nil {
		t.Skip("in some environments .invalid may resolve; skipping")
	}
	if !strings.Contains(err.Error(), "dns resolution failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckUlimitWarning_ZeroConnections(t *testing.T) {
	err := CheckUlimitWarning(0)
	if err != nil {
		t.Errorf("expected no error for 0 connections: %v", err)
	}
}

func TestCheckUlimitWarning_NegativeConnections(t *testing.T) {
	err := CheckUlimitWarning(-1)
	if err != nil {
		t.Errorf("expected no error for negative connections: %v", err)
	}
}

func TestCheckUlimitWarning_ReturnsErrorWhenOverLimit(t *testing.T) {
	// Request an unreasonably high number; on most systems this exceeds RLIMIT_NOFILE
	err := CheckUlimitWarning(10_000_000)
	if err != nil {
		if !strings.Contains(err.Error(), "requested connections") || !strings.Contains(err.Error(), "exceed") {
			t.Errorf("unexpected error message: %v", err)
		}
		return
	}
	// If the system has a very high limit, we get no error; that's acceptable
}

func TestCheckUlimitWarning_SmallRequest(t *testing.T) {
	_ = CheckUlimitWarning(10)
	// On normal systems this does not error; if limit is very low we get an error (acceptable).
}
