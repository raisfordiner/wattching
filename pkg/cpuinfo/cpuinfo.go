package cpuinfo

import (
	"github.com/klauspost/cpuid"
)

type CPUInfo struct {
	VendorString string
	BrandString  string
}

func GetCPUInfo() CPUInfo {
	return CPUInfo{
		VendorString: cpuid.CPU.VendorString,
		BrandString:  cpuid.CPU.BrandName,
	}
}
