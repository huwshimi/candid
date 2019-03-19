// Copyright 2019 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// +build ignore

// Generate certificates for candidtest to use when serving with TLS.

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"math/big"
	"net"
	"os"
	"text/template"
	"time"
)

var output = flag.String("o", "certs.go", "`file` to create")

func main() {
	flag.Parse()
	params, err := generateCerts()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating certificates: %s", err)
		os.Exit(1)
	}
	f, err := os.OpenFile(*output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing certificates: %s", err)
		os.Exit(1)
	}
	if err := genCertTemplate.Execute(f, params); err != nil {
		fmt.Fprintf(os.Stderr, "error writing certificates: %s", err)
		os.Exit(1)
	}
}

func generateCerts() (genCertParams, error) {
	var params genCertParams

	caKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return params, err
	}
	epoch := time.Unix(0, 0)
	caCert := &x509.Certificate{
		SerialNumber: big.NewInt(0),
		Subject: pkix.Name{
			CommonName: "candidtest CA",
		},
		BasicConstraintsValid: true,
		IsCA:                  true,
		NotBefore:             epoch,
		NotAfter:              epoch.Add(1000000 * time.Hour),
	}
	buf, err := x509.CreateCertificate(rand.Reader, caCert, caCert, caKey.Public(), caKey)
	if err != nil {
		return params, err
	}
	params.CACert = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: buf,
	})
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return params, err
	}
	params.Key = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "candidtest",
		},
		NotBefore:   epoch,
		NotAfter:    epoch.Add(1000000 * time.Hour),
		DNSNames:    []string{"localhost", "example.com", "*.example.com"},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}
	buf, err = x509.CreateCertificate(rand.Reader, cert, caCert, key.Public(), caKey)
	if err != nil {
		return params, err
	}
	params.Cert = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: buf,
	})
	return params, nil
}

type genCertParams struct {
	CACert []byte
	Key    []byte
	Cert   []byte
}

var genCertTemplate = template.Must(template.New("").Funcs(template.FuncMap{
	"quote": func(s []byte) string { return "`" + string(s) + "`" },
}).Parse(`
// File generated by generate_certs.go - DO NOT EDIT

package candidtest

var caCert = []byte({{ quote .CACert }})

var key = []byte({{ quote .Key }})

var cert = []byte({{ quote .Cert }})
`[1:]))
