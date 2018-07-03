package proxy

import (
	"crypto/tls"
	"crypto/x509"
	//"encoding/base64"
	"net/http"
	"net/url"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/proxy"
	"k8s.io/client-go/rest"

	"yunion.io/yunioncloud/pkg/httperrors"
	"yunion.io/yunioncloud/pkg/log"

	"yunion.io/yunion-kube/pkg/types/apis"
)

type sErrorResponder struct{}

var (
	er = &sErrorResponder{}
)

func (e *sErrorResponder) Error(w http.ResponseWriter, req *http.Request, err error) {
	httperrors.InternalServerError(w, err.Error())
}

func prefix(cluster *apis.Cluster) string {
	return "/k8s/clusters/" + cluster.Name
}

func New(localConfig *rest.Config, cluster *apis.Cluster) (*SRemoteService, error) {
	return NewLocal(localConfig, cluster)
}

func NewLocal(localConfig *rest.Config, cluster *apis.Cluster) (*SRemoteService, error) {
	hostURL, _, err := rest.DefaultServerURL(localConfig.Host, localConfig.APIPath, schema.GroupVersion{}, true)
	if err != nil {
		return nil, err
	}

	transport, err := rest.TransportFor(localConfig)
	if err != nil {
		return nil, err
	}

	rs := &SRemoteService{
		cluster: cluster,
		url: func() (url.URL, error) {
			return *hostURL, nil
		},
		transport: transport,
	}
	if localConfig.BearerToken != "" {
		rs.auth = "Bearer " + localConfig.BearerToken
	}
	return rs, nil
}

type urlGetter func() (url.URL, error)

type SRemoteService struct {
	cluster   *apis.Cluster
	transport http.RoundTripper
	url       urlGetter
	auth      string
}

func (r *SRemoteService) Close() {}

func (r *SRemoteService) Handler() http.Handler {
	return r
}

func (r *SRemoteService) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	u, err := r.url()
	if err != nil {
		er.Error(rw, req, err)
		return
	}

	log.Infof("=== req url: %q", req.URL.Path)
	u.Path = strings.TrimPrefix(req.URL.Path, "/k8s/clusters/")
	//u.Path = strings.TrimPrefix(req.URL.Path, prefix(r.cluster))
	log.Infof("=== after req url: %q", u.Path)
	u.RawQuery = req.URL.RawQuery
	proto := req.Header.Get("X-Forwarded-Proto")
	if proto != "" {
		req.URL.Scheme = proto
	} else if req.TLS == nil {
		req.URL.Scheme = "http"
	} else {
		req.URL.Scheme = "https"
	}

	req.URL.Host = req.Host
	if r.auth != "" {
		req.Header.Set("Authorization", r.auth)
	}

	httpProxy := proxy.NewUpgradeAwareHandler(&u, r.transport, true, false, er)
	httpProxy.ServeHTTP(rw, req)
}

type SimpleProxy struct {
	url       *url.URL
	transport http.RoundTripper
}

func NewSimpleProxy(host string, caData []byte) (*SimpleProxy, error) {
	hostURL, _, err := rest.DefaultServerURL(host, "", schema.GroupVersion{}, true)
	if err != nil {
		return nil, err
	}

	ht := &http.Transport{}
	if len(caData) > 0 {
		certPool := x509.NewCertPool()
		certPool.AppendCertsFromPEM(caData)
		ht.TLSClientConfig = &tls.Config{
			RootCAs: certPool,
		}
	}

	return &SimpleProxy{
		url:       hostURL,
		transport: ht,
	}, nil
}

func (s *SimpleProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	u := *s.url
	u.Path = req.URL.Path
	u.RawQuery = req.URL.RawQuery
	req.URL.Scheme = "https"
	req.URL.Host = req.Host
	httpProxy := proxy.NewUpgradeAwareHandler(&u, s.transport, true, false, er)
	httpProxy.ServeHTTP(w, req)
}
