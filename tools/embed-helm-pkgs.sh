#!/bin/bash

#helm --namespace monitor template \
    #--name-template monitor \
    #--values ./tools/promtheus-operator-values.yaml \
    #./manifests/prometheus-operator > ./static/prometheus-monitor-template.yaml
rm -rf ./static
helm package ./manifests/helm/monitor-stack \
    --version v1 \
    --destination ./static
helm package ./manifests/helm/fluent-bit \
    --destination ./static
