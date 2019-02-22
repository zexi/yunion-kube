package addons

const YunionManifestTemplate = `
#### CNI plugin ####
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: yunion
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: yunion
subjects:
- kind: ServiceAccount
  name: yunion
  namespace: kube-system
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: yunion
rules:
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - get
  - apiGroups:
      - ""
    resources:
      - nodes
    verbs:
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - nodes/status
    verbs:
      - patch
---
kind: ConfigMap
apiVersion: v1
metadata:
  name: yunion-cni-config
  namespace: kube-system
data:
  cni-conf.json: |
    {
      "cniVersion": "0.3.1",
      "name": "yunion-cni",
      "type": "yunion-bridge",
      "isDefaultGateway": false,
      "cluster_ip_range": "{{.ClusterCIDR}}",
      "ipam": {
        "type": "yunion-ipam",
        "auth_url": "{{.AuthURL}}",
        "admin_user": "{{.AdminUser}}",
        "admin_password": "{{.AdminPassword}}",
        "admin_project": "{{.AdminProject}}",
        "timeout": 30,
        "cluster": "{{.KubeCluster}}",
        "region": "{{.Region}}"
      }
    }
---
kind: DaemonSet
apiVersion: extensions/v1beta1
metadata:
  name: yunion-cni
  namespace: kube-system
  labels:
    k8s-app: yunion-cni
    lxcfs: "false"
spec:
  template:
    metadata:
      labels:
        lxcfs: "false"
        k8s-app: yunion-cni
    spec:
      serviceAccountName: yunion
      hostNetwork: true
      tolerations:
      - operator: Exists
        effect: NoSchedule
      - operator: Exists
        effect: NoExecute
      containers:
        # Runs yunion/cni container on each Kubernetes node.
        # This container installs the Yunion CNI binaries
        # and CNI network config file on each node.
        - name: install-cni
          image: {{.CNIImage}}
          imagePullPolicy: "Always"
          command: ["/install-cni.sh"]
          env:
          # The CNI network config to install on each node.
          - name: CNI_NETWORK_CONFIG
            valueFrom:
              configMapKeyRef:
                name: yunion-cni-config
                key: cni-conf.json
          - name: CNI_CONF_NAME
            value: "10-yunion.conf"
          volumeMounts:
          - mountPath: /host/opt/cni/bin
            name: host-cni-bin
          - mountPath: /host/etc/cni/net.d
            name: host-cni-net
      volumes:
        - name: host-cni-net
          hostPath:
            path: /etc/cni/net.d
        - name: yunion-cni-config
          configMap:
            name: yunion-cni-config
        - name: host-cni-bin
          hostPath:
            path: /opt/cni/bin
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: yunion
  namespace: kube-system
---
####### cloud provider ######
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cloud-controller-manager
  namespace: kube-system
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system:cloud-controller-manager
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: cloud-controller-manager
  namespace: kube-system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    k8s-app: yunion-cloud-controller-manager
  annotations:
     scheduler.alpha.kubernetes.io/critical-pod: ''
  name: yunion-cloud-controller-manager
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      k8s-app: yunion-cloud-controller-manager
  template:
    metadata:
      labels:
        k8s-app: yunion-cloud-controller-manager
        lxcfs: "false"
    spec:
      hostNetwork: true
      serviceAccountName: cloud-controller-manager
      tolerations:
      # this is required so CCM can bootstrap itself
      - key: node.cloudprovider.kubernetes.io/uninitialized
        value: "true"
        effect: NoSchedule
      # cloud controller manages should be able to run on masters
      - key: node-role.kubernetes.io/master
        effect: NoSchedule
      - key: node-role.kubernetes.io/controlplane
        effect: NoSchedule
      # this is to restrict CCM to only run on master nodes
      # the node selector may vary depending on your cluster setup
      containers:
      - name: cloud-controller-manager
        image: {{.CloudProviderImage}}
        command:
        - /yunion-cloud-controller-manager
        - --v=2
        - --leader-elect=true
        - --configure-cloud-routes=false
        - --cloud-config=/etc/kubernetes/cloud-config.json
        - --use-service-account-credentials=false
        volumeMounts:
        - mountPath: /etc/kubernetes
          name: kubernetes-config
      volumes:
      - hostPath:
          path: /etc/kubernetes
          type: Directory
        name: kubernetes-config
---

### Yunion csi StorageClass

apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: csi-yunion
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
provisioner: csi-yunionplugin
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer

---

### Attacher plugin

apiVersion: v1
kind: ServiceAccount
metadata:
  name: csi-attacher
  namespace: kube-system

---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: external-attacher-runner
rules:
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: [""]
    resources: ["persistentvolumes"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["volumeattachments"]
    verbs: ["get", "list", "watch", "update"]

---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: csi-attacher-role
subjects:
  - kind: ServiceAccount
    name: csi-attacher
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: external-attacher-runner
  apiGroup: rbac.authorization.k8s.io

---
kind: Service
apiVersion: v1
metadata:
  name: csi-yunionplugin-attacher
  namespace: kube-system
  labels:
    app: csi-yunionplugin-attacher
spec:
  selector:
    app: csi-yunionplugin-attacher
  ports:
    - name: dummy
      port: 12345

---
kind: StatefulSet
apiVersion: apps/v1beta1
metadata:
  name: csi-yunionplugin-attacher
  namespace: kube-system
spec:
  serviceName: "csi-yunionplugin-attacher"
  replicas: 1
  template:
    metadata:
      labels:
        lxcfs: "false"
        app: csi-yunionplugin-attacher
    spec:
      serviceAccount: csi-attacher
      tolerations:
      # this is required so CCM can bootstrap itself
      - key: node.cloudprovider.kubernetes.io/uninitialized
        value: "true"
        effect: NoSchedule
      # cloud controller manages should be able to run on masters
      - key: node-role.kubernetes.io/master
        effect: NoSchedule
      - key: node-role.kubernetes.io/controlplane
        effect: NoSchedule
      hostNetwork: true
      containers:
        - name: csi-yunionplugin-attacher
          image: {{.CSIAttacher}}
          args:
            - "--v=5"
            - "--csi-address=$(ADDRESS)"
          env:
            - name: ADDRESS
              value: /var/lib/kubelet/plugins/csi-yunionplugin/csi.sock
          imagePullPolicy: "Always"
          volumeMounts:
            - name: socket-dir
              mountPath: /var/lib/kubelet/plugins/csi-yunionplugin
      volumes:
        - name: socket-dir
          hostPath:
            path: /var/lib/kubelet/plugins/csi-yunionplugin
            type: DirectoryOrCreate
---

### Provisioner
apiVersion: v1
kind: ServiceAccount
metadata:
  name: csi-provisioner
  namespace: kube-system

---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: external-provisioner-runner
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list"]
  - apiGroups: [""]
    resources: ["persistentvolumes"]
    verbs: ["get", "list", "watch", "create", "delete"]
  - apiGroups: [""]
    resources: ["persistentvolumeclaims"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["storageclasses"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["list", "watch", "create", "update", "patch"]
  - apiGroups: [""]
    resources: ["endpoints"]
    verbs: ["list", "watch", "create", "update", "get"]
  - apiGroups: ["snapshot.storage.k8s.io"]
    resources: ["volumesnapshots"]
    verbs: ["get", "list"]
  - apiGroups: ["snapshot.storage.k8s.io"]
    resources: ["volumesnapshotcontents"]
    verbs: ["get", "list"]
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get", "list", "update", "patch"]

---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: csi-provisioner-role
subjects:
  - kind: ServiceAccount
    name: csi-provisioner
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: external-provisioner-runner
  apiGroup: rbac.authorization.k8s.io

---
kind: Service
apiVersion: v1
metadata:
  name: csi-yunionplugin-provisioner
  namespace: kube-system
  labels:
    app: csi-yunionplugin-provisioner
spec:
  selector:
    app: csi-yunionplugin-provisioner
  ports:
    - name: dummy
      port: 12345

---
kind: StatefulSet
apiVersion: apps/v1beta1
metadata:
  name: csi-yunionplugin-provisioner
  namespace: kube-system
spec:
  serviceName: "csi-yunionplugin-provisioner"
  replicas: 1
  template:
    metadata:
      labels:
        lxcfs: "false"
        app: csi-yunionplugin-provisioner
    spec:
      serviceAccount: csi-provisioner
      tolerations:
      # this is required so CCM can bootstrap itself
      - key: node.cloudprovider.kubernetes.io/uninitialized
        value: "true"
        effect: NoSchedule
      # cloud controller manages should be able to run on masters
      - key: node-role.kubernetes.io/master
        effect: NoSchedule
      - key: node-role.kubernetes.io/controlplane
        effect: NoSchedule
      hostNetwork: true
      containers:
        - name: csi-provisioner
          image: {{.CSIProvisioner}}
          args:
            - "--provisioner=csi-yunionplugin"
            - "--csi-address=$(ADDRESS)"
            - "--v=5"
            - "--connection-timeout=30s"
            - "--feature-gates=Topology=true"
          env:
            - name: ADDRESS
              value: /var/lib/kubelet/plugins/csi-yunionplugin/csi.sock
          imagePullPolicy: "Always"
          volumeMounts:
            - name: socket-dir
              mountPath: /var/lib/kubelet/plugins/csi-yunionplugin
      volumes:
        - name: socket-dir
          hostPath:
            path: /var/lib/kubelet/plugins/csi-yunionplugin
            type: DirectoryOrCreate

---

### Plugin

apiVersion: v1
kind: ServiceAccount
metadata:
  name: csi-nodeplugin
  namespace: kube-system

---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: csi-nodeplugin
rules:
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get", "list", "update"]
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["get", "list"]
  - apiGroups: [""]
    resources: ["persistentvolumes"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["volumeattachments"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: ["csi.storage.k8s.io"]
    resources: ["csidrivers"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: ["csi.storage.k8s.io"]
    resources: ["csinodeinfos"]
    verbs: ["get", "list", "watch", "update"]

---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: csi-nodeplugin
subjects:
  - kind: ServiceAccount
    name: csi-nodeplugin
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: csi-nodeplugin
  apiGroup: rbac.authorization.k8s.io

---
kind: DaemonSet
apiVersion: apps/v1beta2
metadata:
  name: csi-yunionplugin
  namespace: kube-system
spec:
  selector:
    matchLabels:
      app: csi-yunionplugin
  template:
    metadata:
      labels:
        lxcfs: "false"
        app: csi-yunionplugin
    spec:
      serviceAccount: csi-nodeplugin
      tolerations:
      # this is required so CCM can bootstrap itself
      - key: node.cloudprovider.kubernetes.io/uninitialized
        value: "true"
        effect: NoSchedule
      # cloud controller manages should be able to run on masters
      - key: node-role.kubernetes.io/master
        effect: NoSchedule
      - key: node-role.kubernetes.io/controlplane
        effect: NoSchedule
      hostNetwork: true
      hostPID: true
      # to use e.g. Rook orchestrated cluster, and mons' FQDN is
      # resolved through k8s service, set dns policy to cluster first
      dnsPolicy: ClusterFirstWithHostNet
      containers:
        - name: driver-registrar
          image: {{.CSIRegistrar}}
          args:
            - "--v=5"
            - "--csi-address=$(ADDRESS)"
            - "--mode=node-register"
            #- --pod-info-mount-version="v1"
            - "--kubelet-registration-path=$(DRIVER_REG_SOCK_PATH)"
          env:
            - name: ADDRESS
              value: /var/lib/kubelet/plugins/csi-yunionplugin/csi.sock
            - name: DRIVER_REG_SOCK_PATH
              value: /var/lib/kubelet/plugins/csi-yunionplugin/csi.sock
            - name: KUBE_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
          volumeMounts:
            - name: socket-dir
              mountPath: /var/lib/kubelet/plugins/csi-yunionplugin
            - name: registration-dir
              mountPath: /registration
        - name: csi-yunionplugin
          securityContext:
            privileged: true
            capabilities:
              add: ["SYS_ADMIN"]
            allowPrivilegeEscalation: true
          image: {{.CSIImage}}
          args :
            - "--nodeid=$(NODE_ID)"
            - "--endpoint=$(CSI_ENDPOINT)"
            - "--v=5"
            - "--drivername=csi-yunionplugin"
            - "--authUrl={{.AuthURL}}"
            - "--adminUser={{.AdminUser}}"
            - "--adminPassword={{.AdminPassword}}"
            - "--adminProject={{.AdminProject}}"
            - "--region={{.Region}}"
          env:
            - name: HOST_ROOTFS
              value: "/host"
            - name: NODE_ID
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: CSI_ENDPOINT
              value: unix://var/lib/kubelet/plugins/csi-yunionplugin/csi.sock
          imagePullPolicy: "Always"
          volumeMounts:
            - name: plugin-dir
              mountPath: /var/lib/kubelet/plugins/csi-yunionplugin
            - name: pods-mount-dir
              mountPath: /var/lib/kubelet/pods
              mountPropagation: "Bidirectional"
            - mountPath: /dev
              name: host-dev
            - mountPath: /sys
              name: host-sys
            - mountPath: /lib/modules
              name: lib-modules
              readOnly: true
            - mountPath: /opt/cloud/workspace
              name: yunion-workspace
      volumes:
        - name: plugin-dir
          hostPath:
            path: /var/lib/kubelet/plugins/csi-yunionplugin
            type: DirectoryOrCreate
        - name: registration-dir
          hostPath:
            path: /var/lib/kubelet/plugins/
            type: Directory
        - name: pods-mount-dir
          hostPath:
            path: /var/lib/kubelet/pods
            type: Directory
        - name: socket-dir
          hostPath:
            path: /var/lib/kubelet/plugins/csi-yunionplugin
            type: DirectoryOrCreate
        - name: host-dev
          hostPath:
            path: /dev
        - name: host-sys
          hostPath:
            path: /sys
        - name: lib-modules
          hostPath:
            path: /lib/modules
        - name: yunion-workspace
          hostPath:
            path: /opt/cloud/workspace

#### helm tiller plugin ####
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tiller
  namespace: kube-system

---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: tiller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: tiller
    namespace: kube-system

---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    app: helm
    name: tiller
    lxcfs: "false"
  name: tiller-deploy
  namespace: kube-system
spec:
  template:
    metadata:
      labels:
        app: helm
        name: tiller
        lxcfs: "false"
    spec:
      automountServiceAccountToken: true
      containers:
      - env:
        - name: TILLER_NAMESPACE
          value: kube-system
        - name: TILLER_HISTORY_MAX
          value: "0"
        image: {{ .TillerImage }}
        livenessProbe:
          failureThreshold: 3
          httpGet:
            path: /liveness
            port: 44135
            scheme: HTTP
          initialDelaySeconds: 1
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 1
        name: tiller
        ports:
        - containerPort: 44134
          name: tiller
          protocol: TCP
        - containerPort: 44135
          name: http
          protocol: TCP
        readinessProbe:
          failureThreshold: 3
          httpGet:
            path: /readiness
            port: 44135
            scheme: HTTP
          initialDelaySeconds: 1
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 1
      serviceAccount: tiller
      serviceAccountName: tiller

#### metrics server plugin ####
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: metrics-server:system:auth-delegator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:auth-delegator
subjects:
- kind: ServiceAccount
  name: metrics-server
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: RoleBinding
metadata:
  name: metrics-server-auth-reader
  namespace: kube-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: extension-apiserver-authentication-reader
subjects:
- kind: ServiceAccount
  name: metrics-server
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: system:metrics-server
rules:
- apiGroups:
  - ""
  resources:
  - pods
  - nodes
  - nodes/stats
  - namespaces
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - "extensions"
  resources:
  - deployments
  verbs:
  - get
  - list
  - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:metrics-server
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:metrics-server
subjects:
- kind: ServiceAccount
  name: metrics-server
  namespace: kube-system
---
apiVersion: apiregistration.k8s.io/v1beta1
kind: APIService
metadata:
  name: v1beta1.metrics.k8s.io
spec:
  service:
    name: metrics-server
    namespace: kube-system
  group: metrics.k8s.io
  version: v1beta1
  insecureSkipTLSVerify: true
  groupPriorityMinimum: 100
  versionPriority: 100
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: metrics-server
  namespace: kube-system
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: metrics-server
  namespace: kube-system
  labels:
    k8s-app: metrics-server
    lxcfs: "false"
spec:
  selector:
    matchLabels:
      k8s-app: metrics-server
  template:
    metadata:
      name: metrics-server
      labels:
        k8s-app: metrics-server
        lxcfs: "false"
    spec:
      serviceAccountName: metrics-server
      containers:
      - name: metrics-server
        image: {{ .MetricsServerImage }}
        imagePullPolicy: Always
        command:
        - /metrics-server
        - --kubelet-insecure-tls
        - --kubelet-preferred-address-types=InternalIP
        - --logtostderr
---
apiVersion: v1
kind: Service
metadata:
  name: metrics-server
  namespace: kube-system
  labels:
    kubernetes.io/name: "Metrics-server"
spec:
  selector:
    k8s-app: metrics-server
  ports:
  - port: 443
    protocol: TCP
    targetPort: 443
`
