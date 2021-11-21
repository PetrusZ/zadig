/*
Copyright 2021 The KodeRover Authors.

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

package taskplugin

import (
	"context"
	"time"

	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/koderover/zadig/pkg/microservice/warpdrive/config"
	"github.com/koderover/zadig/pkg/microservice/warpdrive/core/service/types/task"
	krkubeclient "github.com/koderover/zadig/pkg/tool/kube/client"
)

const (
	// TriggerTaskTimeout ...
	TriggerTaskTimeout = 60 * 60 * 1 // 60 minutes
)

// InitializeTriggerTaskPlugin to initialize build task plugin, and return reference
func InitializeTriggerTaskPlugin(taskType config.TaskType) TaskPlugin {
	return &TriggerTaskPlugin{
		Name:       taskType,
		kubeClient: krkubeclient.Client(),
	}
}

// TriggerTaskPlugin is Plugin, name should be compatible with task type
type TriggerTaskPlugin struct {
	Name          config.TaskType
	KubeNamespace string
	JobName       string
	FileName      string
	kubeClient    client.Client
	Task          *task.Trigger
	Log           *zap.SugaredLogger

	ack func()
}

func (p *TriggerTaskPlugin) SetAckFunc(ack func()) {
	p.ack = ack
}

// Init ...
func (p *TriggerTaskPlugin) Init(jobname, filename string, xl *zap.SugaredLogger) {
	p.JobName = jobname
	p.Log = xl
	p.FileName = filename
}

func (p *TriggerTaskPlugin) Type() config.TaskType {
	return p.Name
}

// Status ...
func (p *TriggerTaskPlugin) Status() config.Status {
	return p.Task.TaskStatus
}

// SetStatus ...
func (p *TriggerTaskPlugin) SetStatus(status config.Status) {
	p.Task.TaskStatus = status
}

// TaskTimeout ...
func (p *TriggerTaskPlugin) TaskTimeout() int {
	if p.Task.Timeout == 0 {
		p.Task.Timeout = TriggerTaskTimeout
	} else {
		if !p.Task.IsRestart {
			p.Task.Timeout = p.Task.Timeout * 60
		}
	}
	return p.Task.Timeout
}

func (p *TriggerTaskPlugin) SetTriggerStatusCompleted(status config.Status) {
	p.Task.TaskStatus = status
	p.Task.EndTime = time.Now().Unix()
}

func (p *TriggerTaskPlugin) Run(ctx context.Context, pipelineTask *task.Task, pipelineCtx *task.PipelineCtx, serviceName string) {
	p.Log.Infof("succeed to create trigger task %s", p.JobName)

}

// Wait ...
func (p *TriggerTaskPlugin) Wait(ctx context.Context) {
	timeout := time.After(time.Duration(p.TaskTimeout()))

	for {
		select {
		case <-ctx.Done():
			p.Task.TaskStatus = config.StatusCancelled
			return

		case <-timeout:
			p.Task.TaskStatus = config.StatusTimeout
			return

		default:
			time.Sleep(time.Second * 1)

			if p.IsTaskDone() {
				return
			}
		}
	}
}

// Complete ...
func (p *TriggerTaskPlugin) Complete(ctx context.Context, pipelineTask *task.Task, serviceName string) {
}

// SetTask ...
func (p *TriggerTaskPlugin) SetTask(t map[string]interface{}) error {
	task, err := ToTriggerTask(t)
	if err != nil {
		return err
	}
	p.Task = task
	return nil
}

// GetTask ...
func (p *TriggerTaskPlugin) GetTask() interface{} {
	return p.Task
}

// IsTaskDone ...
func (p *TriggerTaskPlugin) IsTaskDone() bool {
	if p.Task.TaskStatus != config.StatusCreated && p.Task.TaskStatus != config.StatusRunning {
		return true
	}
	return false
}

// IsTaskFailed ...
func (p *TriggerTaskPlugin) IsTaskFailed() bool {
	if p.Task.TaskStatus == config.StatusFailed || p.Task.TaskStatus == config.StatusTimeout || p.Task.TaskStatus == config.StatusCancelled {
		return true
	}
	return false
}

// SetStartTime ...
func (p *TriggerTaskPlugin) SetStartTime() {
	p.Task.StartTime = time.Now().Unix()
}

// SetEndTime ...
func (p *TriggerTaskPlugin) SetEndTime() {
	p.Task.EndTime = time.Now().Unix()
}

// IsTaskEnabled ...
func (p *TriggerTaskPlugin) IsTaskEnabled() bool {
	return p.Task.Enabled
}

// ResetError ...
func (p *TriggerTaskPlugin) ResetError() {
	p.Task.Error = ""
}
