FROM registry.cn-beijing.aliyuncs.com/yunionio/alpine-build:1.0-5 as builder
ARG TARGETPLATFORM
ARG BUILDPLATFORM
RUN mkdir -p /root/go/src/yunion.io/x/yunion-kube
ADD . /root/go/src/yunion.io/x/yunion-kube
WORKDIR /root/go/src/yunion.io/x/yunion-kube
RUN make build

FROM registry.cn-beijing.aliyuncs.com/yunionio/kubectl:1.18.6
RUN mkdir -p /opt/yunion/bin
RUN apk add --no-cache librados librbd && rm -rf /var/cache/apk/*
# COPY --from=builder ./_output/alpine-build/bin/kube-server /opt/yunion/bin/kube-server
COPY --from=builder /root/go/src/yunion.io/x/yunion-kube/_output/bin/ /opt/yunion/bin/
