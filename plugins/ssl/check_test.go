package ssl

import (
	"crypto/x509"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"
)

func TestSignatureAlgorithmName(t *testing.T) {
	tests := []struct {
		algo x509.SignatureAlgorithm
		want string
	}{
		{x509.SHA256WithRSA, "SHA256-RSA"},
		{x509.SHA384WithRSA, "SHA384-RSA"},
		{x509.SHA512WithRSA, "SHA512-RSA"},
		{x509.ECDSAWithSHA256, "ECDSA-SHA256"},
		{x509.ECDSAWithSHA384, "ECDSA-SHA384"},
		{x509.ECDSAWithSHA512, "ECDSA-SHA512"},
		{x509.PureEd25519, "Ed25519"},
		{x509.MD5WithRSA, "MD5-RSA"},
		{x509.SHA1WithRSA, "SHA1-RSA"},
		{x509.SHA256WithRSAPSS, "SHA256-RSAPSS"},
		{x509.SHA384WithRSAPSS, "SHA384-RSAPSS"},
		{x509.SHA512WithRSAPSS, "SHA512-RSAPSS"},
		{x509.DSAWithSHA256, "DSA-SHA256"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := signatureAlgorithmName(tt.algo)
			if got != tt.want {
				t.Errorf("signatureAlgorithmName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIPAddressesToString(t *testing.T) {
	tests := []struct {
		name string
		ips  []net.IP
		want []string
	}{
		{
			name: "空列表",
			ips:  nil,
			want: []string{},
		},
		{
			name: "IPv4",
			ips:  []net.IP{net.ParseIP("192.168.1.1")},
			want: []string{"192.168.1.1"},
		},
		{
			name: "IPv6",
			ips:  []net.IP{net.ParseIP("::1")},
			want: []string{"::1"},
		},
		{
			name: "多个IP",
			ips:  []net.IP{net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.2")},
			want: []string{"10.0.0.1", "10.0.0.2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ipAddressesToString(tt.ips)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d results, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestCertToInfo(t *testing.T) {
	now := time.Now()
	notBefore := now.Add(-24 * time.Hour)     // 昨天
	notAfter := now.Add(180 * 24 * time.Hour) // 180 天后

	cert := &x509.Certificate{
		SerialNumber:       big.NewInt(12345),
		NotBefore:          notBefore,
		NotAfter:           notAfter,
		SignatureAlgorithm: x509.SHA256WithRSA,
		DNSNames:           []string{"example.com", "www.example.com"},
		IPAddresses:        []net.IP{net.ParseIP("1.2.3.4")},
	}
	cert.Subject.CommonName = "example.com"
	cert.Issuer.CommonName = "Let's Encrypt"

	info := certToInfo(cert, 3, "example.com", 443)

	if info.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", info.Domain, "example.com")
	}
	if info.Issuer != "Let's Encrypt" {
		t.Errorf("Issuer = %q, want %q", info.Issuer, "Let's Encrypt")
	}
	if info.Subject != "example.com" {
		t.Errorf("Subject = %q, want %q", info.Subject, "example.com")
	}
	if info.ChainLength != 3 {
		t.Errorf("ChainLength = %d, want %d", info.ChainLength, 3)
	}
	if info.Port != 443 {
		t.Errorf("Port = %d, want %d", info.Port, 443)
	}
	if info.IsExpired {
		t.Error("IsExpired = true, want false")
	}
	if info.DaysUntilExpiry <= 0 {
		t.Errorf("DaysUntilExpiry = %d, want positive", info.DaysUntilExpiry)
	}
	if info.SerialNumber != "3039" {
		t.Errorf("SerialNumber = %q, want %q", info.SerialNumber, "3039")
	}
	if info.SignatureAlgorithm != "SHA256-RSA" {
		t.Errorf("SignatureAlgorithm = %q, want %q", info.SignatureAlgorithm, "SHA256-RSA")
	}
	if len(info.DNSNames) != 2 {
		t.Errorf("DNSNames length = %d, want 2", len(info.DNSNames))
	}
	if len(info.IPAddress) != 1 || info.IPAddress[0] != "1.2.3.4" {
		t.Errorf("IPAddress = %v, want [1.2.3.4]", info.IPAddress)
	}
}

func TestCertToInfo_Expired(t *testing.T) {
	now := time.Now()
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    now.Add(-365 * 24 * time.Hour),
		NotAfter:     now.Add(-10 * 24 * time.Hour), // 10 天前过期
	}
	cert.Subject.CommonName = "expired.com"
	cert.Issuer.CommonName = "Test CA"

	info := certToInfo(cert, 1, "expired.com", 443)

	if !info.IsExpired {
		t.Error("IsExpired = false, want true")
	}
	if info.DaysUntilExpiry >= 0 {
		t.Errorf("DaysUntilExpiry = %d, want negative", info.DaysUntilExpiry)
	}
}

func TestFormatOutput(t *testing.T) {
	info := &CertificateInfo{
		Domain:             "example.com",
		Issuer:             "Let's Encrypt",
		Subject:            "example.com",
		NotBefore:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:           time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		DaysUntilExpiry:    200,
		IsExpired:          false,
		ChainLength:        2,
		SerialNumber:       "ABCD1234",
		SignatureAlgorithm: "SHA256-RSA",
		DNSNames:           []string{"example.com", "www.example.com"},
		Port:               443,
	}

	output := FormatOutput(info, 30)

	checks := []string{
		"example.com",
		"443",
		"Let's Encrypt",
		"2026-01-01 ~ 2027-01-01",
		"✅",
		"SHA256-RSA",
		"example.com, www.example.com",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("FormatOutput missing %q in output:\n%s", check, output)
		}
	}
}

func TestFormatOutput_Expired(t *testing.T) {
	info := &CertificateInfo{
		Domain:          "expired.com",
		Issuer:          "Test CA",
		Subject:         "expired.com",
		NotAfter:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		DaysUntilExpiry: -100,
		IsExpired:       true,
		Port:            443,
	}

	output := FormatOutput(info, 30)
	if !strings.Contains(output, "❌") {
		t.Errorf("FormatOutput for expired cert should contain ❌, got:\n%s", output)
	}
	if !strings.Contains(output, "已过期") {
		t.Errorf("FormatOutput for expired cert should contain '已过期', got:\n%s", output)
	}
}

func TestFormatOutput_Warning(t *testing.T) {
	info := &CertificateInfo{
		Domain:          "warning.com",
		Issuer:          "Test CA",
		Subject:         "warning.com",
		NotAfter:        time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		DaysUntilExpiry: 10,
		IsExpired:       false,
		Port:            443,
	}

	output := FormatOutput(info, 30)
	if !strings.Contains(output, "⚠️") {
		t.Errorf("FormatOutput for nearly-expired cert should contain ⚠️, got:\n%s", output)
	}
	if !strings.Contains(output, "即将过期") {
		t.Errorf("FormatOutput for nearly-expired cert should contain '即将过期', got:\n%s", output)
	}
}

func TestFormatOutput_WithIPAddresses(t *testing.T) {
	info := &CertificateInfo{
		Domain:    "test.com",
		Issuer:    "Test",
		Subject:   "test.com",
		Port:      443,
		IPAddress: []string{"1.2.3.4", "5.6.7.8"},
	}

	output := FormatOutput(info, 30)
	if !strings.Contains(output, "1.2.3.4, 5.6.7.8") {
		t.Errorf("FormatOutput should contain IP addresses, got:\n%s", output)
	}
}

func TestFormatChainOutput(t *testing.T) {
	chain := []*CertificateInfo{
		{
			Domain:          "example.com",
			Subject:         "example.com",
			Issuer:          "R3",
			NotAfter:        time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
			DaysUntilExpiry: 200,
			IsExpired:       false,
			Port:            443,
		},
		{
			Domain:          "[中间证书 1] R3",
			Subject:         "R3",
			Issuer:          "ISRG Root X1",
			NotAfter:        time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC),
			DaysUntilExpiry: 500,
			IsExpired:       false,
			Port:            443,
		},
	}

	output := FormatChainOutput(chain)
	if !strings.Contains(output, "证书链") {
		t.Errorf("FormatChainOutput should contain '证书链', got:\n%s", output)
	}
	if !strings.Contains(output, "R3") {
		t.Errorf("FormatChainOutput should contain 'R3', got:\n%s", output)
	}
	if !strings.Contains(output, "ISRG Root X1") {
		t.Errorf("FormatChainOutput should contain 'ISRG Root X1', got:\n%s", output)
	}
}

func TestFormatChainOutput_Empty(t *testing.T) {
	output := FormatChainOutput(nil)
	if output != "" {
		t.Errorf("FormatChainOutput(nil) = %q, want empty string", output)
	}
}
