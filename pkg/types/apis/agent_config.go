package apis

type AgentConfig struct {
	ServerUrl string `json:"serverUrl"`
	Token     string `json:"token"`
	ClusterId string `json:"clusterId"`
	NodeId    string `json:"nodeId"`
}
