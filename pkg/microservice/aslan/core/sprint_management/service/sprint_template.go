/*
 * Copyright 2023 The KodeRover Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/models"
	commonmodels "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/models"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/mongodb"
	"github.com/koderover/zadig/v2/pkg/shared/handler"
	e "github.com/koderover/zadig/v2/pkg/tool/errors"
	mongotool "github.com/koderover/zadig/v2/pkg/tool/mongo"
	"github.com/koderover/zadig/v2/pkg/types"
	"github.com/koderover/zadig/v2/pkg/util"
)

func CreateSprintTemplate(ctx *handler.Context, args *models.SprintTemplate) error {
	if args.Name == "" {
		return e.ErrCreateSprintTemplate.AddErr(errors.New("Required parameters are missing"))
	}
	err := args.Lint()
	if err != nil {
		return e.ErrCreateSprintTemplate.AddErr(errors.Wrap(err, "Invalid sprint template"))
	}

	args.CreatedBy = ctx.GenUserBriefInfo()
	args.UpdatedBy = ctx.GenUserBriefInfo()
	args.CreateTime = time.Now().Unix()
	args.UpdateTime = time.Now().Unix()
	args.Key, args.KeyInitials = util.GetKeyAndInitials(args.Name)

	err = mongodb.NewSprintTemplateColl().Create(ctx, args)
	if err != nil {
		return e.ErrCreateSprintTemplate.AddErr(errors.Wrap(err, "Create sprint template error"))
	}
	return nil
}

func GetSprintTemplate(ctx *handler.Context, id string) (*models.SprintTemplate, error) {
	sprintTemplate, err := mongodb.NewSprintTemplateColl().Find(ctx, &mongodb.SprintTemplateQueryOption{ID: id})
	if err != nil {
		return nil, e.ErrGetSprintTemplate.AddErr(errors.Wrap(err, "GetSprintTemplate"))
	}

	workflowNames := make([]string, 0)
	workflowMap := make(map[string]*models.SprintWorkflow)
	for _, stage := range sprintTemplate.Stages {
		for _, workflow := range stage.Workflows {
			if !workflow.IsDeleted {
				workflowMap[workflow.Name] = workflow
				workflowNames = append(workflowNames, workflow.Name)
			}
		}
	}

	workflowList, err := mongodb.NewWorkflowColl().List(&mongodb.ListWorkflowOption{Names: workflowNames})
	if err != nil {
		return nil, e.ErrGetSprintTemplate.AddErr(errors.Wrapf(err, "ListWorkflow %s", workflowNames))
	}

	changed := false
	for _, workflow := range workflowList {
		if _, ok := workflowMap[workflow.Name]; !ok {
			return nil, e.ErrGetSprintTemplate.AddErr(errors.Errorf("Workflow %s not found in sprint template", workflow.Name))
		}

		if workflowMap[workflow.Name].DisplayName != workflow.DisplayName {
			workflowMap[workflow.Name].DisplayName = workflow.DisplayName
			changed = true
		}
		delete(workflowMap, workflow.Name)
	}
	if len(workflowMap) > 0 {
		// remind workflows are deleted
		for _, workflow := range workflowMap {
			workflow.IsDeleted = true
		}
		changed = true
	}

	if changed {
		err = mongodb.NewSprintTemplateColl().Update(ctx, sprintTemplate)
		if err != nil {
			return nil, e.ErrGetSprintTemplate.AddErr(errors.Wrapf(err, "Update sprint template %s", sprintTemplate.Name))
		}
	}

	return sprintTemplate, nil
}

func DeleteSprintTemplate(ctx *handler.Context, username, id string) error {
	_, err := mongodb.NewSprintTemplateColl().GetByID(ctx, id)
	if err != nil {
		return e.ErrDeleteSprintTemplate.AddErr(errors.Wrapf(err, "Get sprint template %s", id))
	}
	return mongodb.NewReleasePlanColl().DeleteByID(ctx, id)
}

func UpdateSprintTemplate(ctx *handler.Context, sprintTemplateID string, args *models.SprintTemplate) error {
	session, deferFunc, err := mongotool.SessionWithTransaction(ctx)
	defer func() { deferFunc(err) }()
	if err != nil {
		return e.ErrUpdateSprintTemplate.AddErr(errors.Wrap(err, "SessionWithTransaction"))
	}

	err = args.Lint()
	if err != nil {
		return e.ErrUpdateSprintTemplate.AddErr(errors.Wrap(err, "该流程不合法"))
	}

	template, err := mongodb.NewSprintTemplateColl().GetByID(ctx, sprintTemplateID)
	if err != nil {
		return e.ErrUpdateSprintTemplate.AddErr(errors.Wrap(err, "Get sprint template"))
	}

	if template.UpdateTime > args.UpdateTime {
		return e.ErrUpdateSprintTemplate.AddErr(errors.New("该流程已被他人更新，请刷新后重试"))
	}

	template.Stages = args.Stages
	template.UpdatedBy = ctx.GenUserBriefInfo()
	template.UpdateTime = time.Now().Unix()

	sprints, _, err := mongodb.NewSprintCollWithSession(session).List(ctx, &mongodb.ListSprintOption{TemplateID: sprintTemplateID})
	if err != nil {
		return e.ErrUpdateSprintTemplate.AddErr(errors.Wrapf(err, "List sprints by template id %s", sprintTemplateID))
	}

	if len(sprints) > 0 {
		stageTemplateMap := make(map[string]*models.SprintStageTemplate)
		for _, stage := range template.Stages {
			stageTemplateMap[stage.ID] = stage
		}

		for _, sprint := range sprints {
			stagesMap := make(map[string]*models.SprintStage)
			for _, stage := range sprint.Stages {
				if stageTemplateMap[stage.ID] == nil {
					// not find this stage in template, delete stage if it's empty
					if len(stage.WorkItemIDs) != 0 {
						return e.ErrUpdateSprintTemplate.AddErr(errors.Errorf("\n由于「%s」的「%s」中包含工作项，所以无法删除，如需删除，请先移除相应工作项", sprint.Name, stage.Name))
					}
				}

				stagesMap[stage.ID] = stage
			}

			newStages := make([]*models.SprintStage, 0)
			for _, stageTemplate := range template.Stages {
				if stageTemplate.ID == "" {
					stageTemplate.ID = uuid.NewString()
				}

				stageTemplateWorkflowMap := make(map[string]*models.SprintWorkflow)
				for _, workflow := range stageTemplate.Workflows {
					stageTemplateWorkflowMap[workflow.Name] = workflow
				}

				if stage, ok := stagesMap[stageTemplate.ID]; !ok {
					// not find this stage in sprint, add new stage
					newStages = append(newStages, &models.SprintStage{
						ID:        stageTemplate.ID,
						Name:      stageTemplate.Name,
						Workflows: stageTemplate.Workflows,
					})
				} else {
					// find stage in sprint, update it
					stage.Name = stageTemplate.Name

					stageWorkflowMap := make(map[string]*models.SprintWorkflow)
					// set deleted workflows to IsDeleted = true
					for _, workflow := range stage.Workflows {
						if _, ok := stageTemplateWorkflowMap[workflow.Name]; !ok {
							workflow.IsDeleted = true
						}
						stageWorkflowMap[workflow.Name] = workflow
					}

					newWorkflows := make([]*models.SprintWorkflow, 0)
					// add/update workflows in stage template
					for _, workflowTemplate := range stageTemplate.Workflows {
						if workflow, ok := stageWorkflowMap[workflowTemplate.Name]; !ok {
							// not find this workflow in stage, add new workflow
							newWorkflows = append(newWorkflows, workflowTemplate)
						} else {
							// find this workflow in stage, update it
							workflow.DisplayName = workflowTemplate.DisplayName
							workflow.IsDeleted = workflowTemplate.IsDeleted
							newWorkflows = append(newWorkflows, workflow)
							delete(stageWorkflowMap, workflow.Name)
						}
					}

					// add remind workflows in stage
					for _, workflow := range stageWorkflowMap {
						newWorkflows = append(newWorkflows, workflow)
					}

					stage.Workflows = newWorkflows

					newStages = append(newStages, stage)
				}
			}

			sprint.UpdatedBy = ctx.GenUserBriefInfo()
			sprint.UpdateTime = time.Now().Unix()
			sprint.Stages = newStages
		}

		err = mongodb.NewSprintCollWithSession(session).BulkUpdateStages(ctx, sprints)
		if err != nil {
			return e.ErrUpdateSprintTemplate.AddErr(errors.Wrap(err, "Bulk Update Sprints Stages"))
		}
	}

	if err = mongodb.NewSprintTemplateCollWithSession(session).Update(ctx, template); err != nil {
		return e.ErrUpdateSprintTemplate.AddErr(errors.Wrap(err, "Update sprint template"))
	}

	return nil
}

type ListSprintTemplateOption struct {
	ProjectName string `form:"projectName" binding:"required"`
	PageNum     int64  `form:"pageNum" binding:"required"`
	PageSize    int64  `form:"pageSize" binding:"required"`
}

type ListSprintTemplateResp struct {
	List  []*models.SprintTemplate `json:"list"`
	Total int64                    `json:"total"`
}

func ListSprintTemplate(ctx *handler.Context, opt *ListSprintTemplateOption) (*ListSprintTemplateResp, error) {
	var (
		list  []*commonmodels.SprintTemplate
		total int64
		err   error
	)
	list, err = mongodb.NewSprintTemplateColl().List(ctx, &mongodb.SprintTemplateListOption{ProjectName: opt.ProjectName})
	if err != nil {
		return nil, e.ErrListSprintTemplate.AddErr(errors.Wrap(err, "ListSprintTemplate"))
	}

	return &ListSprintTemplateResp{
		List:  list,
		Total: total,
	}, nil
}

func InitSprintTemplate(ctx *handler.Context, projectName string) {
	template := &models.SprintTemplate{
		Name: "zadig-default",
		Stages: []*models.SprintStageTemplate{
			{ID: uuid.NewString(), Name: "开发阶段"},
			{ID: uuid.NewString(), Name: "测试阶段"},
			{ID: uuid.NewString(), Name: "预发阶段"},
			{ID: uuid.NewString(), Name: "生产阶段"},
		},
	}
	template.ProjectName = projectName
	template.CreateTime = time.Now().Unix()
	template.UpdateTime = time.Now().Unix()
	template.CreatedBy = types.GeneSystemUserBriefInfo()
	template.UpdatedBy = types.GeneSystemUserBriefInfo()

	err := template.Lint()
	if err != nil {
		ctx.Logger.Errorf("Invalid sprint template: %v", err)
		return
	}

	if err := mongodb.NewSprintTemplateColl().UpsertByName(ctx, template); err != nil {
		ctx.Logger.Errorf("Update build-in sprint template error: %v", err)
	}
}
