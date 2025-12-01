//go:build windows

package action

var baseLocation = getenvOrDefault("USERPROFILE", `c:\\`)
