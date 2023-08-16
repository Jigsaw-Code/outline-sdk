package httpproxy

import (
	"crypto/rsa"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"crypto/rand"
	"net"
	"sort"
	"sync"
	"time"
)

// CaSigner is a certificate signer by CA certificate. It supports caching.
type CaSigner struct {
	// Ca specifies CA certificate. You must set before using.
	Ca *tls.Certificate

	mu        sync.RWMutex
	certMap   map[string]*tls.Certificate
	certList  []string
	certIndex int
	certMax   int
}

// NewCaSigner returns a new CaSigner without caching.
func NewCaSigner() *CaSigner {
	return NewCaSignerCache(0)
}

// NewCaSignerCache returns a new CaSigner with caching given max.
func NewCaSignerCache(max int) *CaSigner {
	if max < 0 {
		max = 0
	}
	return &CaSigner{
		certMap:   make(map[string]*tls.Certificate),
		certList:  make([]string, max),
		certIndex: 0,
		certMax:   max,
	}
}

// SignHost generates TLS certificate given single host, signed by CA certificate.
func (c *CaSigner) SignHost(host string) (cert *tls.Certificate) {
	if host == "" {
		return
	}

	if c.certMax <= 0 {
		crt, err := SignHosts(*c.Ca, []string{host})
		if err != nil {
			return nil
		}
		cert = crt
		return
	}

	func() {
		c.mu.RLock()
		defer c.mu.RUnlock()
		cert = c.certMap[host]
	}()

	if cert != nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	cert = c.certMap[host]
	if cert != nil {
		return
	}

	crt, err := SignHosts(*c.Ca, []string{host})
	if err != nil {
		return nil
	}

	cert = crt
	if len(c.certMap) >= c.certMax {
		delete(c.certMap, c.certList[c.certIndex])
	}

	c.certMap[host] = cert
	c.certList[c.certIndex] = host
	c.certIndex++
	if c.certIndex >= c.certMax {
		c.certIndex = 0
	}

	return
}

// SignHosts generates TLS certificate given hosts, signed by CA certificate.
func SignHosts(ca tls.Certificate, hosts []string) (*tls.Certificate, error) {
	x509ca, err := x509.ParseCertificate(ca.Certificate[0])
	if err != nil {
		return nil, err
	}

	start := time.Unix(0, 0)
	end, _ := time.Parse("2006-01-02", "2038-01-19")
	serial := hashSortedBigInt(append(hosts, "1"))
	template := x509.Certificate{
		SerialNumber:          serial,
		Issuer:                x509ca.Subject,
		Subject:               x509ca.Subject,
		NotBefore:             start,
		NotAfter:              end,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	for _, h := range hosts {
		h = stripPort(h)
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	certPriv, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, x509ca, &certPriv.PublicKey, ca.PrivateKey)
	if err != nil {
		return nil, err
	}

	return &tls.Certificate{
		Certificate: [][]byte{derBytes, ca.Certificate[0]},
		PrivateKey:  certPriv,
	}, nil
}

func hashSorted(lst []string) []byte {
	c := make([]string, len(lst))
	copy(c, lst)
	sort.Strings(c)
	h := sha1.New()

	for _, s := range c {
		h.Write([]byte(s + ","))
	}

	return h.Sum(nil)
}

func hashSortedBigInt(lst []string) *big.Int {
	rv := new(big.Int)
	rv.SetBytes(hashSorted(lst))

	return rv
}
