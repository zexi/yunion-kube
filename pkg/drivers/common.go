package drivers

import (
	"yunion.io/x/yunion-kube/pkg/models/types"
)

func GetControlplaneMachineDatas(clusterId string, data []*types.CreateMachineData) ([]*types.CreateMachineData, []*types.CreateMachineData) {
	controls := make([]*types.CreateMachineData, 0)
	nodes := make([]*types.CreateMachineData, 0)
	for _, d := range data {
		if len(clusterId) != 0 {
			d.ClusterId = clusterId
		}
		if d.Role == types.RoleTypeControlplane {
			controls = append(controls, d)
		} else {
			nodes = append(nodes, d)
		}
	}
	return controls, nodes
}
