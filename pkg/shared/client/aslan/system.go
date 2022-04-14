/*
Copyright 2022 The KodeRover Authors.

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

package aslan

import (
	"github.com/koderover/zadig/pkg/tool/httpclient"
)

type CleanConfig struct {
	Cron        string `json:"cron"`
	CronEnabled bool   `json:"cron_enabled"`
}

func (c *Client) GetDockerCleanConfig() (*CleanConfig, error) {
	url := "/system/cleanCache/state"

	res := new(CleanConfig)

	_, err := c.Get(url, httpclient.SetResult(res))
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Client) DockerClean() error {
	url := "/system/cleanCache/oneClick"
	_, err := c.Post(url)
	return err
}
