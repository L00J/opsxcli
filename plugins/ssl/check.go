package ssl

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"strings"
	"time"
)

// CertificateInfo SSL 证书信息
type CertificateInfo struct {
	Domain             string
	Issuer             string
	Subject            string
	NotBefore          time.Time
	NotAfter           time.Time
	DaysUntilExpiry    int
	IsExpired          bool
	ChainLength        int
	SerialNumber       string
	SignatureAlgorithm string
	DNSNames           []string
	IPAddress          []string
	Port               int
}

// CheckOptions SSL 检查选项
type CheckOptions struct {
	Domain    string
	Port      int
	Timeout   time.Duration
	ShowChain bool
	WarnDays  int
}

// Check 连接 TLS 获取证书信息
func Check(opts *CheckOptions) (*CertificateInfo, error) {
	if opts.Port == 0 {
		opts.Port = 443
	}
	if opts.Timeout == 0 {
		opts.Timeout = 10 * time.Second
	}
	if opts.WarnDays == 0 {
		opts.WarnDays = 30
	}

	addr := fmt.Sprintf("%s:%d", opts.Domain, opts.Port)

	dialer := &net.Dialer{
		Timeout: opts.Timeout,
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         opts.Domain,
	})
	if err != nil {
		return nil, fmt.Errorf("连接 %s 失败: %w", addr, err)
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil, fmt.Errorf("未获取到证书")
	}

	return certToInfo(certs[0], len(certs), opts.Domain, opts.Port), nil
}

// CheckChain 获取完整证书链
func CheckChain(opts *CheckOptions) ([]*CertificateInfo, error) {
	if opts.Port == 0 {
		opts.Port = 443
	}
	if opts.Timeout == 0 {
		opts.Timeout = 10 * time.Second
	}

	addr := fmt.Sprintf("%s:%d", opts.Domain, opts.Port)

	dialer := &net.Dialer{
		Timeout: opts.Timeout,
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         opts.Domain,
	})
	if err != nil {
		return nil, fmt.Errorf("连接 %s 失败: %w", addr, err)
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil, fmt.Errorf("未获取到证书")
	}

	chain := make([]*CertificateInfo, 0, len(certs))
	for i, cert := range certs {
		info := certToInfo(cert, len(certs), opts.Domain, opts.Port)
		// 标记证书链中的层级
		if i > 0 {
			info.Domain = fmt.Sprintf("[中间证书 %d] %s", i, info.Subject)
		}
		chain = append(chain, info)
	}

	return chain, nil
}

// certToInfo 将 x509 证书转换为 CertificateInfo
func certToInfo(cert *x509.Certificate, chainLen int, domain string, port int) *CertificateInfo {
	now := time.Now()
	daysUntil := int(cert.NotAfter.Sub(now).Hours() / 24)

	return &CertificateInfo{
		Domain:             domain,
		Issuer:             cert.Issuer.CommonName,
		Subject:            cert.Subject.CommonName,
		NotBefore:          cert.NotBefore,
		NotAfter:           cert.NotAfter,
		DaysUntilExpiry:    daysUntil,
		IsExpired:          now.After(cert.NotAfter),
		ChainLength:        chainLen,
		SerialNumber:       fmt.Sprintf("%X", cert.SerialNumber),
		SignatureAlgorithm: signatureAlgorithmName(cert.SignatureAlgorithm),
		DNSNames:           cert.DNSNames,
		IPAddress:          ipAddressesToString(cert.IPAddresses),
		Port:               port,
	}
}

// signatureAlgorithmName 获取签名算法的可读名称
func signatureAlgorithmName(algo x509.SignatureAlgorithm) string {
	switch algo {
	case x509.MD2WithRSA:
		return "MD2-RSA"
	case x509.MD5WithRSA:
		return "MD5-RSA"
	case x509.SHA1WithRSA:
		return "SHA1-RSA"
	case x509.SHA256WithRSA:
		return "SHA256-RSA"
	case x509.SHA384WithRSA:
		return "SHA384-RSA"
	case x509.SHA512WithRSA:
		return "SHA512-RSA"
	case x509.DSAWithSHA1:
		return "DSA-SHA1"
	case x509.DSAWithSHA256:
		return "DSA-SHA256"
	case x509.ECDSAWithSHA1:
		return "ECDSA-SHA1"
	case x509.ECDSAWithSHA256:
		return "ECDSA-SHA256"
	case x509.ECDSAWithSHA384:
		return "ECDSA-SHA384"
	case x509.ECDSAWithSHA512:
		return "ECDSA-SHA512"
	case x509.SHA256WithRSAPSS:
		return "SHA256-RSAPSS"
	case x509.SHA384WithRSAPSS:
		return "SHA384-RSAPSS"
	case x509.SHA512WithRSAPSS:
		return "SHA512-RSAPSS"
	case x509.PureEd25519:
		return "Ed25519"
	default:
		return algo.String()
	}
}

// ipAddressesToString 将 net.IP 列表转换为字符串列表
func ipAddressesToString(ips []net.IP) []string {
	result := make([]string, 0, len(ips))
	for _, ip := range ips {
		result = append(result, ip.String())
	}
	return result
}

// FormatOutput 格式化证书信息输出（彩色终端输出）
func FormatOutput(info *CertificateInfo, warnDays int) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("🔍 SSL 证书检查: %s:%d\n", info.Domain, info.Port))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	sb.WriteString(fmt.Sprintf("  域名:        %s\n", info.Subject))
	sb.WriteString(fmt.Sprintf("  签发者:      %s\n", info.Issuer))
	sb.WriteString(fmt.Sprintf("  有效期:      %s ~ %s\n",
		info.NotBefore.Format("2006-01-02"),
		info.NotAfter.Format("2006-01-02"),
	))

	// 剩余天数和状态
	if info.IsExpired {
		sb.WriteString(fmt.Sprintf("  剩余天数:    %d 天（已过期 %d 天）\n",
			info.DaysUntilExpiry, -info.DaysUntilExpiry))
		sb.WriteString("  状态:        ❌ 已过期\n")
	} else if info.DaysUntilExpiry <= warnDays {
		sb.WriteString(fmt.Sprintf("  剩余天数:    %d 天\n", info.DaysUntilExpiry))
		sb.WriteString(fmt.Sprintf("  状态:        ⚠️ 即将过期（%d 天内到期）\n", warnDays))
	} else {
		sb.WriteString(fmt.Sprintf("  剩余天数:    %d 天\n", info.DaysUntilExpiry))
		sb.WriteString("  状态:        ✅ 有效\n")
	}

	sb.WriteString(fmt.Sprintf("  序列号:      %s\n", info.SerialNumber))
	sb.WriteString(fmt.Sprintf("  签名算法:    %s\n", info.SignatureAlgorithm))

	// SANs (DNS 名称)
	if len(info.DNSNames) > 0 {
		sb.WriteString(fmt.Sprintf("  SANs:        %s\n", strings.Join(info.DNSNames, ", ")))
	}

	// IP 地址
	if len(info.IPAddress) > 0 {
		sb.WriteString(fmt.Sprintf("  IP地址:      %s\n", strings.Join(info.IPAddress, ", ")))
	}

	return sb.String()
}

// FormatChainOutput 格式化证书链输出
func FormatChainOutput(chain []*CertificateInfo) string {
	if len(chain) == 0 {
		return ""
	}

	var sb strings.Builder

	// 输出叶子证书（第一个）
	sb.WriteString(FormatOutput(chain[0], 30))

	if len(chain) > 1 {
		sb.WriteString(fmt.Sprintf("\n🔗 证书链 (%d 层):\n", len(chain)))
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		for i, cert := range chain {
			status := "✅"
			if cert.IsExpired {
				status = "❌"
			} else if cert.DaysUntilExpiry <= 30 {
				status = "⚠️"
			}
			sb.WriteString(fmt.Sprintf("  [%d] %s %s (签发者: %s, 有效期至 %s)\n",
				i+1, status, cert.Subject, cert.Issuer,
				cert.NotAfter.Format("2006-01-02"),
			))
		}
	}

	return sb.String()
}
