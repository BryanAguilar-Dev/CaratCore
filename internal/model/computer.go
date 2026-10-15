package model

type Computer struct {
	Hostname              string
	HostnameError         string
	OperatingSystem       string
	Distribution          string
	DistributionVersion   string
	Architecture          string
	PlatformSupported     bool
	DockerCLIAvailable    bool
	DockerEngineAvailable bool
	DockerVersion         string
	DockerError           string
}
