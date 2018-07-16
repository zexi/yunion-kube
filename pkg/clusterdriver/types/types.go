package types

type ClusterInfo struct {
	Version             string
	ServiceAccountToken string
	Endpoint            string
	RootCaCertificate   string
	ClientCertificate   string
	ClientKey           string
	NodeCount           int64
	Metadata            map[string]string
	Status              string
}

type DriverOptions struct {
	BoolOptions   map[string]bool
	StringOptions map[string]string
	IntOptions    map[string]int64
}
