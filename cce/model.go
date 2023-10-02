package cce

type KubeConfigParams struct {
	ProjectName    string
	ClusterName    string
	DaysValid      string
	TargetLocation string
	Server         string
}

type KubeConfig struct {
	Kind        string   `json:"kind"`
	APIVersion  string   `json:"apiVersion"`
	Preferences struct{} `json:"preferences"`
	Clusters    []struct {
		Name    string `json:"name"`
		Cluster struct {
			Server                   string `json:"server"`
			CertificateAuthorityData string `json:"certificate-authority-data,omitempty"`
			InsecureSkipTLSVerify    bool   `json:"insecure-skip-tls-verify,omitempty"`
		} `json:"cluster,omitempty"`
	} `json:"clusters"`
	Users []struct {
		Name string `json:"name"`
		User struct {
			ClientCertificateData string `json:"client-certificate-data"`
			ClientKeyData         string `json:"client-key-data"`
		} `json:"user"`
	} `json:"users"`
	Contexts []struct {
		Name    string `json:"name"`
		Context struct {
			Cluster string `json:"cluster"`
			User    string `json:"user"`
		} `json:"context"`
	} `json:"contexts"`
	CurrentContext string `json:"current-context"`
}
