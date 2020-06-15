module yunion.io/x/yunion-kube

go 1.12

require (
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/Masterminds/semver/v3 v3.0.1
	github.com/Masterminds/sprig v2.16.0+incompatible
	github.com/ceph/go-ceph v0.0.0-20181217221554-e32f9f0f2e94
	github.com/docker/docker v1.4.2-0.20181221150755-2cb26cfe9cbf
	github.com/ghodss/yaml v1.0.0
	github.com/go-sql-driver/mysql v1.4.1
	github.com/gofrs/flock v0.7.1
	github.com/golang/protobuf v1.3.2
	github.com/gorilla/mux v1.7.0
	github.com/gorilla/websocket v1.4.0
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2
	github.com/zexi/golosetup v0.0.0-20181117053200-8c308e8bbf44
	go.etcd.io/etcd v0.5.0-alpha.5.0.20191023171146-3cf2f69b5738
	golang.org/x/net v0.0.0-20191028085509-fe3aa8a45271
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	google.golang.org/grpc v1.24.0
	gopkg.in/urfave/cli.v1 v1.20.0
	gopkg.in/yaml.v2 v2.2.4
	helm.sh/helm/v3 v3.0.0
	k8s.io/api v0.0.0
	k8s.io/apimachinery v0.0.0
	k8s.io/apiserver v0.0.0
	k8s.io/cli-runtime v0.0.0
	k8s.io/client-go v9.0.0+incompatible
	k8s.io/cluster-bootstrap v0.0.0
	k8s.io/gengo v0.0.0-20200425085600-19394052f0fa
	k8s.io/helm v2.12.3+incompatible
	k8s.io/klog v1.0.0
	k8s.io/kube-proxy v0.0.0
	k8s.io/kubectl v0.0.0
	k8s.io/kubernetes v1.16.0
	sigs.k8s.io/controller-runtime v0.1.10
	sigs.k8s.io/yaml v1.1.0
	yunion.io/x/code-generator v0.0.0-20200525091704-bb27d6630736
	yunion.io/x/jsonutils v0.0.0-20200615014624-f9c3576579c9
	yunion.io/x/log v0.0.0-20200313080802-57a4ce5966b3
	yunion.io/x/onecloud v0.0.0-20200612020938-20efee53f262
	yunion.io/x/pkg v0.0.0-20200615071345-60a252beb982
	yunion.io/x/sqlchemy v0.0.0-20200608080702-9b6683aa048c
)

replace (
	github.com/docker/docker => github.com/docker/docker v0.0.0-20190731150326-928381b2215c
	github.com/renstrom/dedent => github.com/lithammer/dedent v1.1.0
	github.com/ugorji/go => github.com/ugorji/go v1.1.2
	helm.sh/helm/v3 => helm.sh/helm/v3 v3.0.0

	k8s.io/api => k8s.io/api v0.0.0-20190918155943-95b840bb6a1f

	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190918161926-8f644eb6e783

	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655

	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190918160949-bfa5e2e684ad

	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190918162238-f783a3654da8

	k8s.io/client-go => k8s.io/client-go v0.0.0-20190918160344-1fbdaa4c8d90

	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190918163234-a9c1f33e9fb9

	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20190918163108-da9fdfce26bb

	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190912054826-cd179ad6a269

	k8s.io/component-base => k8s.io/component-base v0.0.0-20190918160511-547f6c5d7090

	k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190828162817-608eb1dad4ac

	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20190918163402-db86a8c7bb21

	k8s.io/gengo => github.com/zexi/gengo v0.0.0-20200425085600-19394052f0fa

	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20190918161219-8c8f079fddc3

	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20190918162944-7a93a0ddadd8

	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20190918162534-de037b596c1e

	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20190918162820-3b5c1246eb18

	k8s.io/kubectl => k8s.io/kubectl v0.0.0-20190918164019-21692a0861df

	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20190918162654-250a1838aa2c

	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20190918163543-cfa506e53441

	k8s.io/metrics => k8s.io/metrics v0.0.0-20190918162108-227c654b2546

	k8s.io/node-api => k8s.io/node-api v0.0.0-20190918163711-2299658ad911

	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20190918161442-d4c9c65c82af

	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.0.0-20190918162410-e45c26d066f2

	k8s.io/sample-controller => k8s.io/sample-controller v0.0.0-20190918161628-92eb3cb7496c
)
