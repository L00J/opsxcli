package ssl

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// createTestTLSServer 创建一个自签名 TLS 测试服务器
func createTestTLSServer(t *testing.T) *httptest.Server {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("生成私钥失败: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(42),
		Subject: pkix.Name{
			CommonName: "Test CA",
		},
		NotBefore:   time.Now().Add(-24 * time.Hour),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{"localhost"},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("创建证书失败: %v", err)
	}

	cert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  privateKey,
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	server.TLS = &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	server.StartTLS()
	return server
}

// TestCheck_WithTestServer 测试 Check 连接 TLS 服务器获取证书信息
func TestCheck_WithTestServer(t *testing.T) {
	server := createTestTLSServer(t)
	defer server.Close()

	hostPort := strings.TrimPrefix(server.URL, "https://")
	host, portStr, _ := net.SplitHostPort(hostPort)
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	opts := &CheckOptions{
		Domain:  host,
		Port:    port,
		Timeout: 5 * time.Second,
	}

	info, err := Check(opts)
	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}

	if info.Domain != host {
		t.Errorf("Domain = %q, want %q", info.Domain, host)
	}
	if info.Port != port {
		t.Errorf("Port = %d, want %d", info.Port, port)
	}
	if info.ChainLength < 1 {
		t.Errorf("ChainLength = %d, want >= 1", info.ChainLength)
	}
	if info.IsExpired {
		t.Error("IsExpired = true, want false")
	}
	if info.SerialNumber == "" {
		t.Error("SerialNumber is empty")
	}
}

// TestCheck_DefaultPort 测试默认端口和超时设置
func TestCheck_DefaultPort(t *testing.T) {
	server := createTestTLSServer(t)
	defer server.Close()

	hostPort := strings.TrimPrefix(server.URL, "https://")
	host, portStr, _ := net.SplitHostPort(hostPort)
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	opts := &CheckOptions{
		Domain: host,
		Port:   port,
	}
	// 不设置 Timeout 和 WarnDays，测试默认值
	info, err := Check(opts)
	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}
	if opts.Port != port {
		t.Errorf("Port = %d, want %d (should not be changed)", opts.Port, port)
	}
	if info == nil {
		t.Fatal("info is nil")
	}
}

// TestCheckChain_WithTestServer 测试 CheckChain 获取完整证书链
func TestCheckChain_WithTestServer(t *testing.T) {
	server := createTestTLSServer(t)
	defer server.Close()

	hostPort := strings.TrimPrefix(server.URL, "https://")
	host, portStr, _ := net.SplitHostPort(hostPort)
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	opts := &CheckOptions{
		Domain:  host,
		Port:    port,
		Timeout: 5 * time.Second,
	}

	chain, err := CheckChain(opts)
	if err != nil {
		t.Fatalf("CheckChain() error: %v", err)
	}

	if len(chain) < 1 {
		t.Fatalf("Chain length = %d, want >= 1", len(chain))
	}

	// 第一个证书应该是叶子证书
	if chain[0].Port != port {
		t.Errorf("Chain[0].Port = %d, want %d", chain[0].Port, port)
	}
}

// TestCheck_InvalidHost 测试连接无效主机返回错误
func TestCheck_InvalidHost(t *testing.T) {
	opts := &CheckOptions{
		Domain:  "invalid.host.that.does.not.exist.example",
		Port:    443,
		Timeout: 2 * time.Second,
	}

	_, err := Check(opts)
	if err == nil {
		t.Fatal("expected error for invalid host")
	}
	if !strings.Contains(err.Error(), "连接") {
		t.Errorf("error should contain '连接', got: %v", err)
	}
}

// TestCheckChain_InvalidHost 测试 CheckChain 连接失败
func TestCheckChain_InvalidHost(t *testing.T) {
	opts := &CheckOptions{
		Domain:  "invalid.host.that.does.not.exist.example",
		Port:    443,
		Timeout: 2 * time.Second,
	}

	_, err := CheckChain(opts)
	if err == nil {
		t.Fatal("expected error for invalid host")
	}
}

// TestCheck_SetsDefaultWarnDays 测试默认 WarnDays 设置
func TestCheck_SetsDefaultWarnDays(t *testing.T) {
	server := createTestTLSServer(t)
	defer server.Close()

	hostPort := strings.TrimPrefix(server.URL, "https://")
	host, portStr, _ := net.SplitHostPort(hostPort)
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	opts := &CheckOptions{
		Domain: host,
		Port:   port,
	}

	_, err := Check(opts)
	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}
	if opts.WarnDays != 30 {
		t.Errorf("WarnDays = %d, want 30 (default)", opts.WarnDays)
	}
}

// TestCheckChain_DefaultPort 测试 CheckChain 默认端口
func TestCheckChain_DefaultPort(t *testing.T) {
	server := createTestTLSServer(t)
	defer server.Close()

	hostPort := strings.TrimPrefix(server.URL, "https://")
	host, portStr, _ := net.SplitHostPort(hostPort)
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	opts := &CheckOptions{
		Domain: host,
		Port:   port,
	}

	chain, err := CheckChain(opts)
	if err != nil {
		t.Fatalf("CheckChain() error: %v", err)
	}
	if len(chain) < 1 {
		t.Fatal("chain is empty")
	}
}

// TestFormatChainOutput_WithExpiredCert 测试证书链中有过期证书
func TestFormatChainOutput_WithExpiredCert(t *testing.T) {
	chain := []*CertificateInfo{
		{
			Domain:          "expired.com",
			Subject:         "expired.com",
			Issuer:          "Test CA",
			NotAfter:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			DaysUntilExpiry: -100,
			IsExpired:       true,
			Port:            443,
		},
		{
			Domain:          "[中间证书 1] Test CA",
			Subject:         "Test CA",
			Issuer:          "Root CA",
			NotAfter:        time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
			DaysUntilExpiry: 500,
			IsExpired:       false,
			Port:            443,
		},
	}

	output := FormatChainOutput(chain)
	if !strings.Contains(output, "❌") {
		t.Errorf("should contain ❌ for expired cert, got:\n%s", output)
	}
	if !strings.Contains(output, "✅") {
		t.Errorf("should contain ✅ for valid cert, got:\n%s", output)
	}
	if !strings.Contains(output, "证书链") {
		t.Errorf("should contain '证书链', got:\n%s", output)
	}
}

// TestFormatChainOutput_SingleCert 测试单证书链（无中间证书）
func TestFormatChainOutput_SingleCert(t *testing.T) {
	chain := []*CertificateInfo{
		{
			Domain:          "single.com",
			Subject:         "single.com",
			Issuer:          "Root CA",
			NotAfter:        time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
			DaysUntilExpiry: 200,
			IsExpired:       false,
			Port:            443,
		},
	}

	output := FormatChainOutput(chain)
	// 单证书不应该显示"证书链"标题
	if strings.Contains(output, "证书链") {
		t.Errorf("single cert chain should not contain '证书链', got:\n%s", output)
	}
	if !strings.Contains(output, "single.com") {
		t.Errorf("should contain domain name, got:\n%s", output)
	}
}

// TestSignatureAlgorithmName_Default 测试未知签名算法返回默认值
func TestSignatureAlgorithmName_Default(t *testing.T) {
	result := signatureAlgorithmName(x509.UnknownSignatureAlgorithm)
	if result == "" {
		t.Error("expected non-empty result for unknown algorithm")
	}
}

// TestSignatureAlgorithmName_Uncommon 测试不常见的签名算法
func TestSignatureAlgorithmName_Uncommon(t *testing.T) {
	tests := []struct {
		algo x509.SignatureAlgorithm
		want string
	}{
		{x509.MD2WithRSA, "MD2-RSA"},
		{x509.DSAWithSHA1, "DSA-SHA1"},
		{x509.ECDSAWithSHA1, "ECDSA-SHA1"},
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
