package modules

type LoadbalancerManager struct {
	ResourceManager
}

var (
	Loadbalancers LoadbalancerManager
)

func init() {
	Loadbalancers = LoadbalancerManager{
		NewComputeManager(
			"loadbalancer",
			"loadbalancers",
			[]string{
				"id",
				"name",
				"status",
				"address_type",
				"address",
				"network_type",
				"network_id",
				"zone_id",
			},
			[]string{"tenant"},
		),
	}
	registerCompute(&Loadbalancers)

}
