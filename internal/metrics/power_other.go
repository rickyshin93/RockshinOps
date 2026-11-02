//go:build !darwin

package metrics

import "rockshinops/internal/localization"

func readThermal() Thermal {
	return Thermal{Error: localization.T("当前平台尚未实现热压力采集")}
}

func readPowerAssertions() PowerAssertions {
	return PowerAssertions{Items: []PowerAssertion{}, Error: localization.T("当前平台尚未实现电源断言采集")}
}
