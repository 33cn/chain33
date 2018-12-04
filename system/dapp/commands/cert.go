package commands

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

//CertCmd generate cert
func CertCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cert",
		Short: "generate cert",
		Run: func(cmd *cobra.Command, args []string) {
			host, _ := cmd.Flags().GetString("host")
			validFrom, _ := cmd.Flags().GetString("start-date")
			ecdsaCurve, _ := cmd.Flags().GetString("ecdsa-curve")
			rsaBits, _ := cmd.Flags().GetInt("rsa-bits")
			isCA, _ := cmd.Flags().GetBool("ca")
			validFor, _ := cmd.Flags().GetDuration("duration")
			certGenerate(host, validFrom, ecdsaCurve, rsaBits, isCA, validFor)
		},
	}
	addCertFlags(cmd)
	return cmd
}

func addCertFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("host", "", "", "Comma-separated hostnames and IPs to generate a certificate for")
	cmd.Flags().StringP("start-date", "", "", "Creation date formatted as Jan 1 15:04:05 2011")
	cmd.Flags().StringP("ecdsa-curve", "", "", "ECDSA curve to use to generate a key. Valid values are P224, P256 (recommended), P384, P521")
	cmd.Flags().IntP("rsa-bits", "", 2048, "Size of RSA key to generate. Ignored if --ecdsa-curve is set")
	cmd.Flags().Bool("ca", false, "whether this cert should be its own Certificate Authority")
	cmd.Flags().Duration("duration", 365*24*time.Hour, "Duration that certificate is valid for")
	cmd.MarkFlagRequired("host")
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
			os.Exit(2)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}

func certGenerate(host, validFrom, ecdsaCurve string, rsaBits int, isCA bool, validFor time.Duration) {
	var priv interface{}
	var err error
	switch ecdsaCurve {
	case "":
		priv, err = rsa.GenerateKey(rand.Reader, rsaBits)
	case "P224":
		priv, err = ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	case "P256":
		priv, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case "P384":
		priv, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	case "P521":
		priv, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	default:
		fmt.Fprintf(os.Stderr, "Unrecognized elliptic curve: %q", ecdsaCurve)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate private key: %s\n", err)
		os.Exit(1)
	}

	var notBefore time.Time
	if len(validFrom) == 0 {
		notBefore = time.Now()
	} else {
		notBefore, err = time.Parse("Jan 2 15:04:05 2006", validFrom)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse creation date: %s\n", err)
			os.Exit(1)
		}
	}

	notAfter := notBefore.Add(validFor)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate serial number: %s", err)
		os.Exit(1)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	hosts := strings.Split(host, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	if isCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create certificate: %s", err)
		os.Exit(1)
	}

	certOut, err := os.Create("cert.pem")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open cert.pem for writing: %s", err)
		os.Exit(1)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write data to cert.pem: %s", err)
		os.Exit(1)
	}
	if err := certOut.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "error closing cert.pem: %s", err)
		os.Exit(1)
	}
	fmt.Print("wrote cert.pem\n")

	keyOut, err := os.OpenFile("key.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open key.pem for writing: %s", err)
		os.Exit(1)
	}
	if err := pem.Encode(keyOut, pemBlockForKey(priv)); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write data to key.pem: %s", err)
		os.Exit(1)
	}
	if err := keyOut.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "error closing key.pem: %s", err)
		os.Exit(1)
	}
	fmt.Print("wrote key.pem\n")
}
