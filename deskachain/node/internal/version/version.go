package version

import (
	"fmt"
	"runtime"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

type InfoData struct {
	AppName   string `json:"app_name"`
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	OSArch    string `json:"os_arch"`
	Networks  string `json:"networks"`
	Mainnet   string `json:"mainnet"`
}

func Info(appName string) InfoData {
	return InfoData{
		AppName:   appName,
		Version:   fallback(Version, "dev"),
		Commit:    fallback(Commit, "unknown"),
		BuildDate: fallback(BuildDate, "unknown"),
		GoVersion: runtime.Version(),
		OSArch:    runtime.GOOS + "/" + runtime.GOARCH,
		Networks:  "localnet,testnet",
		Mainnet:   "not available",
	}
}

func String(appName string) string {
	info := Info(appName)
	return fmt.Sprintf("%s\nversion: %s\ncommit: %s\nbuilt: %s\ngo: %s\nos/arch: %s\nnetworks: %s\nmainnet: %s\n",
		info.AppName,
		info.Version,
		info.Commit,
		info.BuildDate,
		info.GoVersion,
		info.OSArch,
		info.Networks,
		info.Mainnet,
	)
}

func UserAgent(appName string) string {
	info := Info(appName)
	return appName + "/" + info.Version
}

func fallback(value, def string) string {
	if value == "" {
		return def
	}
	return value
}
