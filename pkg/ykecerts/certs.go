package ykecerts

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"io/ioutil"
	"os"
	"path/filepath"

	"k8s.io/client-go/util/cert"

	"yunion.io/x/pkg/util/errors"
	"yunion.io/x/yke/pkg/pki"
	"yunion.io/x/yke/pkg/types"

	"yunion.io/x/yunion-kube/pkg/clusterdriver/yke/ykecerts"
	"yunion.io/x/yunion-kube/pkg/libyke"
)

const (
	bundleFile = "./management-state/certs/bundle.json"
)

type Bundle struct {
	certs map[string]pki.CertificatePKI
}

func newBundle(certs map[string]pki.CertificatePKI) *Bundle {
	return &Bundle{
		certs: certs,
	}
}

func Unmarshal(input string) (*Bundle, error) {
	certs, err := ykecerts.LoadString(input)
	return newBundle(certs), err
}

func (b *Bundle) Certs() map[string]pki.CertificatePKI {
	return b.certs
}

func LoadLocal() (*Bundle, error) {
	f, err := os.Open(bundleFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	certMap, err := ykecerts.Load(f)
	if err != nil {
		return nil, err
	}
	return newBundle(certMap), nil
}

func Generate(config *types.KubernetesEngineConfig) (*Bundle, error) {
	certs, err := libyke.New().GenerateCerts(config)
	if err != nil {
		return nil, err
	}

	return &Bundle{
		certs: certs,
	}, nil
}

func (b *Bundle) Marshal() (string, error) {
	output := &bytes.Buffer{}
	err := ykecerts.Save(b.certs, output)
	return output.String(), err
}

func (b *Bundle) ForNode(config *types.KubernetesEngineConfig, nodeAddress string) *Bundle {
	certs := libyke.New().GenerateYKENodeCerts(context.Background(), *config, nodeAddress, b.certs)
	return &Bundle{
		certs: certs,
	}
}

func (b *Bundle) SaveLocal() error {
	bundlePath := filepath.Dir(bundleFile)
	if err := os.MkdirAll(bundlePath, 0700); err != nil {
		return err
	}

	f, err := ioutil.TempFile(bundlePath, "bundle-")
	if err != nil {
		return err
	}
	defer f.Close()
	defer os.Remove(f.Name())

	if err := ykecerts.Save(b.certs, f); err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	return os.Rename(f.Name(), bundleFile)
}

func (b *Bundle) KubeConfig() string {
	return b.certs["kube-admin"].ConfigPath
}

func (b *Bundle) Explode() error {
	f := &fileWriter{}
	for _, item := range b.certs {
		f.write(item.Path, nil, item.Certificate, nil)
		f.write(item.ConfigPath, []byte(item.Config), nil, nil)
		f.write(item.KeyPath, nil, nil, item.Key)
	}

	return f.err()
}

type fileWriter struct {
	errs []error
}

func (f *fileWriter) write(path string, content []byte, x509cert *x509.Certificate, key *rsa.PrivateKey) {
	if x509cert != nil {
		content = cert.EncodeCertPEM(x509cert)
	}

	if key != nil {
		content = cert.EncodePrivateKeyPEM(key)
	}

	if path == "" || len(content) == 0 {
		return
	}

	existing, err := ioutil.ReadFile(path)
	if err == nil && bytes.Equal(existing, content) {
		return
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		f.errs = append(f.errs, err)
	}
	if err := ioutil.WriteFile(path, content, 0600); err != nil {
		f.errs = append(f.errs, err)
	}
}

func (f *fileWriter) err() error {
	return errors.NewAggregate(f.errs)
}
