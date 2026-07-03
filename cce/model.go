package cce

type KubeConfigParams struct {
	ProjectName    string
	ClusterName    string
	DaysValid      string
	TargetLocation string
	Server         string
}

type cceClusterItem struct {
	Metadata struct {
		Name string `json:"name"`
		UID  string `json:"uid"`
	} `json:"metadata"`
}
