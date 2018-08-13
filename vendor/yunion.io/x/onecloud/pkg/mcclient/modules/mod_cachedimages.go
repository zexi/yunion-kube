package modules

var (
	Cachedimages ResourceManager
)

func init() {
	Cachedimages = NewComputeManager("cachedimage", "cachedimages",
		[]string{"ID", "Name", "Size", "Format", "Owner", "OS_Type", "OS_Distribution", "OS_version", "Hypervisor", "Host_count", "Status"},
		[]string{})

	registerCompute(&Cachedimages)
}
