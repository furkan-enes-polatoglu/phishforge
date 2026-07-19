// Package dkim provides DKIM key generation and message signing. DKIM is a
// standard, legitimate email-authentication mechanism that improves deliverability
// for authorized senders; this is not a spam-filter evasion technique.
package dkim

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"

	xdkim "github.com/emersion/go-msgauth/dkim"
)

// GenerateKey creates a 2048-bit RSA keypair and returns the PEM-encoded private
// key (to store on the sending profile) and the DNS TXT record value to publish
// at <selector>._domainkey.<domain>.
func GenerateKey(selector, domain string) (privatePEM string, dnsRecordName string, dnsRecordValue string, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", "", err
	}
	privDER := x509.MarshalPKCS1PrivateKey(key)
	privatePEM = string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privDER}))

	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", "", "", err
	}
	pubB64 := base64.StdEncoding.EncodeToString(pubDER)
	dnsRecordName = fmt.Sprintf("%s._domainkey.%s", selector, domain)
	dnsRecordValue = "v=DKIM1; k=rsa; p=" + pubB64
	return privatePEM, dnsRecordName, dnsRecordValue, nil
}

// parsePrivateKey accepts a PEM private key in PKCS#1 or PKCS#8 form.
func parsePrivateKey(privatePEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privatePEM))
	if block == nil {
		return nil, fmt.Errorf("invalid PEM private key")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaKey, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA private key")
	}
	return rsaKey, nil
}

// Sign DKIM-signs a raw RFC5322 message and returns the signed message. CRLF line
// endings are ensured because DKIM canonicalization requires them.
func Sign(rawMessage, domain, selector, privatePEM string) (string, error) {
	key, err := parsePrivateKey(privatePEM)
	if err != nil {
		return "", err
	}
	normalized := strings.ReplaceAll(strings.ReplaceAll(rawMessage, "\r\n", "\n"), "\n", "\r\n")

	opts := &xdkim.SignOptions{
		Domain:   domain,
		Selector: selector,
		Signer:   key,
		HeaderKeys: []string{"From", "To", "Subject", "Date", "MIME-Version", "Content-Type"},
	}
	var buf bytes.Buffer
	if err := xdkim.Sign(&buf, strings.NewReader(normalized), opts); err != nil {
		return "", err
	}
	return buf.String(), nil
}
