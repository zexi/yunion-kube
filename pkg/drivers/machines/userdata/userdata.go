/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package userdata

import (
	"bytes"
	"text/template"

	"github.com/pkg/errors"
)

const (
	defaultHeader = `#!/usr/bin/env bash

# Copyright 2018 by the contributors
#
#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.

set -o verbose
set -o errexit
set -o nounset
set -o pipefail
`

	dockerScript = `
configure_docker() {
	cat >/etc/docker/daemon.json <<EOF
{
    "graph": "/opt/docker",
    "registry-mirrors": [
        "https://lje6zxpk.mirror.aliyuncs.com",
        "https://lms7sxqp.mirror.aliyuncs.com",
        "https://registry.docker-cn.com"
    ],
    "insecure-registries": [],
    "live-restore": true
}
EOF

	cat >/etc/logrotate.d/docker-container <<EOF
/opt/docker/containers/*/*.log {
    rotate 5
    daily
    missingok
    dateext
    copytruncate
    notifempty
    compress
    size 10M
}
EOF
	 systemctl enable docker
	 systemctl restart docker
}
configure_docker`

	onecloudConfig = `

mkdir -p /etc/kubernetes

cat >/etc/kubernetes/cloud-config.json <<EOF
{
  "auth_url": "{{.AuthURL}}",
  "admin_user": "{{.AdminUser}}",
  "admin_password": "{{.AdminPassword}}",
  "admin_project": "{{.AdminProject}}",
  "region": "{{.Region}}",
  "cluster": "{{.Cluster}}"
}
EOF

cat >/etc/kubernetes/k8s-sched-policy.json <<EOF
{
    "kind": "Policy",
    "apiVersion": "v1",
    "extenders": [
       {
           "urlPrefix": "{{.SchedulerEndpoint}}/k8s",
           "apiVersion": "v1beta1",
           "filterVerb": "predicates",
           "bindVerb": "",
           "prioritizeVerb": "",
           "weight": 1,
           "enableHttps": true,
           "tlsConfig": {"insecure": true},
           "nodeCacheCapable": false,
           "httpTimeout": 10000000000
        }
    ]
}
EOF
`
)

type baseUserData struct {
	Header         string
	DockerScript   string
	OnecloudConfig string
}

func newBaseUserData(conf BaseConfigure) (*baseUserData, error) {
	var err error
	data := new(baseUserData)
	data.Header = defaultHeader
	data.DockerScript = dockerScript
	data.OnecloudConfig, err = generate("onecloudConfig", onecloudConfig, conf.OnecloudConfigure)
	if err != nil {
		return nil, err
	}
	return data, nil
}

type BaseConfigure struct {
	DockerConfigure
	OnecloudConfigure
}

type DockerConfigure struct {
	DockerGraphDir string
	DockerBIP      string
}

type OnecloudConfigure struct {
	AuthURL           string
	AdminUser         string
	AdminPassword     string
	AdminProject      string
	Region            string
	Cluster           string
	SchedulerEndpoint string
}

func generate(kind string, tpl string, data interface{}) (string, error) {
	t, err := template.New(kind).Parse(tpl)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse %s template", kind)
	}

	var out bytes.Buffer
	if err := t.Execute(&out, data); err != nil {
		return "", errors.Wrapf(err, "failed to generate %s template", kind)
	}

	return out.String(), nil
}
