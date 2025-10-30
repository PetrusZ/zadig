/*
Copyright 2023 The KodeRover Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gin

import (
	"github.com/gin-gonic/gin"
)

const (
	ErrorLicenseMissing             = "未填写许可证"
	ErrorCodeLicenseMissing         = 1000
	ErrorLicenseInvalidVersion      = "许可证版本不匹配"
	ErrorCodeLicenseInvalidVersion  = 1002
	ErrorLicenseExpired             = "许可证已过期"
	ErrorCodeLicenseExpired         = 1001
	ErrorLicenseInvalidSystemID     = "许可证系统不匹配"
	ErrorCodeLicenseInvalidSystemID = 1003
	ErrorUnknown                    = "未知错误"
	ErrorCodeUnknown                = 1010
)

const (
	ZadigXLicenseStatusUninitialized   = "uninitialized"
	ZadigXLicenseStatusNormal          = "normal"
	ZadigXLicenseStatusExpired         = "expired"
	ZadigXLicenseStatusVersionMismatch = "version_mismatch"
	ZadigXLicenseStatusInvalidSystem   = "invalid_system"
)

// TODO: the way we process the license need to be changed after
func ProcessLicense() gin.HandlerFunc {
	return func(c *gin.Context) {
		// if not enterprise we skip
		c.Next()
		return
	}
}
