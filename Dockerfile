FROM registry.cn-beijing.aliyuncs.com/yunionio/centos-build:1.0-1 as builder
ADD . /root/go/src/yunion.io/x/yunion-kube
WORKDIR /root/go/src/yunion.io/x/yunion-kube
RUN make build bundle

FROM yunion/kubectl:1.14.3
RUN mkdir -p /opt/yunion/bin
COPY --from=builder /root/go/src/yunion.io/x/yunion-kube/_output/bin/kube-server /opt/yunion/bin/kube-server
COPY --from=builder /root/go/src/yunion.io/x/yunion-kube/_output/bin/.kube-server.bin /opt/yunion/bin/.kube-server.bin
COPY --from=builder /root/go/src/yunion.io/x/yunion-kube/_output/bin/bundles/kube-server /opt/yunion/bin/bundles/kube-server
