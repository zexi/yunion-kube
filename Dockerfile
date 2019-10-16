FROM yunion/kubectl:1.14.3

MAINTAINER "Zexi Li <lizexi@yunionyun.com>"

RUN mkdir -p /opt/yunion/bin

ADD ./_output/bin/kube-server /opt/yunion/bin/kube-server
