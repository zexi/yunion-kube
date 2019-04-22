package userdata

const (
	nodeCloudInit = `{{.Header}}
write_files:
-   path: /tmp/kubeadm-node.yaml
    owner: root:root
    permissions: '0640'
    content: |
      ---
{{.JoinConfiguration | Indent 6}}
kubeadm:
  operation: join
  config: /tmp/kubeadm-node.yaml
`
)

// NodeInputCloudInit defines the context to generate a node user data
type NodeInputCloudInit struct {
	baseUserDataCloudInit

	JoinConfiguration string
}

// NewNodeCloudInit returns the user data string to be used on a node instance
func NewNodeCloudInit(input *NodeInputCloudInit) (string, error) {
	input.Header = cloudConfigHeader
	fMap := map[string]interface{}{
		"Indent": templateYAMLIndent,
	}
	return generateWithFuncs("node", nodeCloudInit, fMap, input)
}
