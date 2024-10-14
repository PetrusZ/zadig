/*
 * Copyright 2024 The KodeRover Authors.
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
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/koderover/zadig/v2/pkg/microservice/aslan/config"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/models"
	commonmodels "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/models"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/mongodb"
	workflowservice "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/workflow/service/workflow"
	"github.com/koderover/zadig/v2/pkg/setting"
	"github.com/koderover/zadig/v2/pkg/shared/handler"
	e "github.com/koderover/zadig/v2/pkg/tool/errors"
	mongotool "github.com/koderover/zadig/v2/pkg/tool/mongo"
	"github.com/koderover/zadig/v2/pkg/types"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateSprintWorkItem(ctx *handler.Context, args *models.SprintWorkItem) error {
	if args.Title == "" || args.SprintID == "" || args.StageID == "" {
		return e.ErrCreateSprintWorkItem.AddErr(errors.New("Required parameters are missing"))
	}
	args.CreateTime = time.Now().Unix()
	args.UpdateTime = time.Now().Unix()

	lock := getSprintLock(args.SprintID)
	lock.Lock()
	defer lock.Unlock()

	session, deferFunc, err := mongotool.SessionWithTransaction(ctx)
	defer func() { deferFunc(err) }()
	if err != nil {
		return e.ErrCreateSprintWorkItem.AddErr(errors.Wrap(err, "SessionWithTransaction"))
	}

	id, err := mongodb.NewSprintWorkItemCollWithSession(session).Create(ctx, args)
	if err != nil {
		return e.ErrCreateSprintWorkItem.AddErr(errors.Wrap(err, "Create sprint"))
	}

	sprint, err := mongodb.NewSprintCollWithSession(session).GetByID(ctx, args.SprintID)
	if err != nil {
		return e.ErrCreateSprintWorkItem.AddErr(errors.Wrapf(err, "Get sprint by id %s", args.SprintID))
	}

	found := false
	for _, stage := range sprint.Stages {
		if args.StageID == stage.ID {
			stage.WorkItemIDs = append(stage.WorkItemIDs, id.Hex())
			found = true
			break
		}
	}
	if !found {
		return e.ErrCreateSprintWorkItem.AddErr(fmt.Errorf("Stage %s not found", args.StageID))
	}

	err = mongodb.NewSprintCollWithSession(session).UpdateStages(ctx, args.SprintID, sprint.Stages)
	if err != nil {
		return e.ErrCreateSprintWorkItem.AddErr(errors.Wrapf(err, "Update sprint %s stages", args.SprintID))
	}

	activityContent := fmt.Sprintf("创建工作项")
	err = createSprintWorkItemActivity(ctx, session, id.Hex(), setting.SprintWorkItemActivityTypeEvent, activityContent)
	if err != nil {
		return e.ErrCreateSprintWorkItem.AddErr(errors.Wrap(err, "Create sprint workitem activity"))
	}

	return nil
}

type SprintWorkItem struct {
	ID          primitive.ObjectID               `json:"id"`
	Title       string                           `json:"title"`
	Description string                           `json:"description"`
	Owners      []types.UserBriefInfo            `json:"owners"`
	StageID     string                           `json:"stage_id"`
	CreateTime  int64                            `json:"create_time"`
	UpdateTime  int64                            `json:"update_time"`
	Activities  []*models.SprintWorkItemActivity `json:"activities"`
}

func GetSprintWorkItem(ctx *handler.Context, id string) (*SprintWorkItem, error) {
	workitem, err := mongodb.NewSprintWorkItemColl().GetByID(ctx, id)
	if err != nil {
		return nil, e.ErrGetSprintWorkItem.AddErr(errors.Wrapf(err, "Get workitems by sprint id %s", id))
	}

	resp := &SprintWorkItem{
		ID:          workitem.ID,
		Title:       workitem.Title,
		Description: workitem.Description,
		Owners:      workitem.Owners,
		StageID:     workitem.StageID,
		CreateTime:  workitem.CreateTime,
		UpdateTime:  workitem.UpdateTime,
	}

	actives, err := mongodb.NewSprintWorkItemActivityColl().GetByWorkItemID(ctx, id)
	if err != nil {
		return nil, e.ErrGetSprintWorkItem.AddErr(errors.Wrapf(err, "Get activities by workitem id %s", id))
	}
	resp.Activities = actives

	return resp, nil
}

func DeleteSprintWorkItem(ctx *handler.Context, id string) error {
	workitem, err := mongodb.NewSprintWorkItemColl().GetByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "Get sprint")
	}

	lock := getSprintLock(workitem.SprintID)
	lock.Lock()
	defer lock.Unlock()

	session, deferFunc, err := mongotool.SessionWithTransaction(ctx)
	defer func() { deferFunc(err) }()
	if err != nil {
		return e.ErrDeleteSprintWorkItem.AddErr(errors.Wrap(err, "SessionWithTransaction"))
	}

	sprint, err := mongodb.NewSprintCollWithSession(session).GetByID(ctx, workitem.SprintID)
	if err != nil {
		return e.ErrDeleteSprintWorkItem.AddErr(errors.Wrapf(err, "Get sprint by id %s", workitem.SprintID))
	}
	for _, stage := range sprint.Stages {
		for i, workitemID := range stage.WorkItemIDs {
			if workitemID == id {
				stage.WorkItemIDs = append(stage.WorkItemIDs[:i], stage.WorkItemIDs[i:]...)
				break
			}
		}
	}

	err = mongodb.NewSprintWorkItemColl().DeleteByID(ctx, id)
	if err != nil {
		return e.ErrDeleteSprintWorkItem.AddErr(errors.Wrap(err, "Delete sprint workitem"))
	}

	return nil
}

func UpdateSprintWorkItemTitle(ctx *handler.Context, id, title string) error {
	session, deferFunc, err := mongotool.SessionWithTransaction(ctx)
	defer func() { deferFunc(err) }()
	if err != nil {
		return e.ErrUpdateSprintWorkItemTitle.AddErr(errors.Wrap(err, "SessionWithTransaction"))
	}

	workitem, err := mongodb.NewSprintWorkItemCollWithSession(session).GetByID(ctx, id)
	if err != nil {
		return e.ErrUpdateSprintWorkItemTitle.AddErr(errors.Wrapf(err, "Get workitems by sprint id %s", id))
	}

	err = mongodb.NewSprintWorkItemCollWithSession(session).UpdateTitle(ctx, id, title)
	if err != nil {
		return e.ErrUpdateSprintWorkItemTitle.AddErr(errors.Wrapf(err, "Update sprint workitem %s title", id))
	}

	activityContent := fmt.Sprintf("重新命名工作项标题：（原为 %s）", workitem.Title)
	err = createSprintWorkItemActivity(ctx, session, id, setting.SprintWorkItemActivityTypeEvent, activityContent)
	if err != nil {
		return e.ErrUpdateSprintWorkItemTitle.AddErr(errors.Wrap(err, "Create sprint workitem activity"))
	}

	return nil
}

func UpdateSprintWorkItemDescpition(ctx *handler.Context, id, description string) error {
	err := mongodb.NewSprintWorkItemColl().UpdateDescription(ctx, id, description)
	if err != nil {
		return e.ErrUpdateSprintWorkItemDesc.AddErr(errors.Wrapf(err, "Update sprint workitem %s description", id))
	}

	return nil
}

func MoveSprintWorkItem(ctx *handler.Context, id, stageID string, index int, sprintUpdateTime, workItemUpdateTime int64) error {
	lock := getSprintLock(id)
	lock.Lock()
	defer lock.Unlock()

	session, deferFunc, err := mongotool.SessionWithTransaction(ctx)
	defer func() { deferFunc(err) }()
	if err != nil {
		return e.ErrMoveSprintWorkItem.AddErr(errors.Wrap(err, "SessionWithTransaction"))
	}

	workitem, err := mongodb.NewSprintWorkItemCollWithSession(session).GetByID(ctx, id)
	if err != nil {
		return e.ErrMoveSprintWorkItem.AddErr(errors.Wrapf(err, "Get workitems by sprint id %s", id))
	}
	if workitem.UpdateTime > workItemUpdateTime {
		return e.ErrMoveSprintWorkItem.AddErr(errors.New("迭代工作项已被其他人更新，请刷新后重试"))
	}

	sprint, err := mongodb.NewSprintCollWithSession(session).GetByID(ctx, workitem.SprintID)
	if err != nil {
		return e.ErrMoveSprintWorkItem.AddErr(errors.Wrapf(err, "Get sprint by id %s", workitem.SprintID))
	}
	// if sprint.UpdateTime > sprintUpdateTime {
	// 	return e.ErrMoveSprintWorkItem.AddErr(errors.New("迭代已被其他人更新，请刷新后重试"))
	// }

	// update sprint stage's WorkItemIDs
	if workitem.StageID == stageID {
		// move to the same stage
		for _, stage := range sprint.Stages {
			if index > len(stage.WorkItemIDs)+1 {
				return e.ErrMoveSprintWorkItem.AddErr(fmt.Errorf("Invalid stage index, requset %d but size is %d", index, len(stage.WorkItemIDs)+1))
			}
			if index == len(stage.WorkItemIDs)+1 {
				// move workitem to here, insert id
				stage.WorkItemIDs = append(stage.WorkItemIDs, id)
			} else {
				newWorkItemIDs := make([]string, 0)
				for i, origID := range stage.WorkItemIDs {
					if i == index {
						// move workitem to here, insert id
						newWorkItemIDs = append(newWorkItemIDs, id)
					} else if origID != id {
						newWorkItemIDs = append(newWorkItemIDs, origID)
					}
				}
				stage.WorkItemIDs = newWorkItemIDs
			}
		}
	} else {
		// move to different stage

		// remove workitem from old stage
		for _, stage := range sprint.Stages {
			if stage.Name == workitem.StageID {
				newWorkItemIDs := make([]string, 0)
				for _, origID := range stage.WorkItemIDs {
					if origID != id {
						newWorkItemIDs = append(newWorkItemIDs, origID)
					}
				}
				stage.WorkItemIDs = newWorkItemIDs
				break
			}
		}
		// add workitem to new stage
		for _, stage := range sprint.Stages {
			if stage.Name == stageID {
				if index > len(stage.WorkItemIDs)+1 {
					return e.ErrMoveSprintWorkItem.AddErr(fmt.Errorf("Invalid stage index, requset %d but size is %d", index, len(stage.WorkItemIDs)+1))
				}
				if index == len(stage.WorkItemIDs)+1 {
					// move workitem to here, insert id
					stage.WorkItemIDs = append(stage.WorkItemIDs, id)
				} else {
					newWorkItemIDs := make([]string, 0)
					for i, origID := range stage.WorkItemIDs {
						if i == index {
							// move workitem to here, insert id
							newWorkItemIDs = append(newWorkItemIDs, id)
						}
						newWorkItemIDs = append(newWorkItemIDs, origID)
					}
					stage.WorkItemIDs = newWorkItemIDs
				}
				break
			}
		}
	}

	err = mongodb.NewSprintCollWithSession(session).UpdateStages(ctx, id, sprint.Stages)
	if err != nil {
		return e.ErrMoveSprintWorkItem.AddErr(errors.Wrapf(err, "Update sprint workitem %s title", id))
	}

	err = mongodb.NewSprintWorkItemCollWithSession(session).UpdateStage(ctx, id, stageID)
	if err != nil {
		return e.ErrMoveSprintWorkItem.AddErr(errors.Wrapf(err, "Update sprint workitem %s title", id))
	}

	if workitem.StageID != stageID {
		activityContent := fmt.Sprintf("移动工作项：从「%s」到「%s」", workitem.StageID, stageID)
		err = createSprintWorkItemActivity(ctx, session, id, setting.SprintWorkItemActivityTypeEvent, activityContent)
		if err != nil {
			return e.ErrMoveSprintWorkItem.AddErr(errors.Wrap(err, "Create sprint workitem activity"))
		}
	}

	return nil
}

type UpdateSprintWorkItemOwnersVerb string

const (
	UpdateSprintWorkItemOwnersVerbAdd    UpdateSprintWorkItemOwnersVerb = "add"
	UpdateSprintWorkItemOwnersVerbRemove UpdateSprintWorkItemOwnersVerb = "remove"
)

func UpdateSprintWorkItemOwners(ctx *handler.Context, id, verb string, owners []types.UserBriefInfo) error {
	session, deferFunc, err := mongotool.SessionWithTransaction(ctx)
	defer func() { deferFunc(err) }()
	if err != nil {
		return e.ErrUpdateSprintWorkItemOwner.AddErr(errors.Wrap(err, "SessionWithTransaction"))
	}

	workitem, err := mongodb.NewSprintWorkItemCollWithSession(session).GetByID(ctx, id)
	if err != nil {
		return e.ErrUpdateSprintWorkItemOwner.AddErr(errors.Wrapf(err, "Get workitems by sprint id %s", id))
	}

	ownersMap := make(map[string]types.UserBriefInfo)
	for _, owner := range workitem.Owners {
		ownersMap[owner.UID] = owner
	}

	activityContent := fmt.Sprintf("修改负责人：")
	switch UpdateSprintWorkItemOwnersVerb(verb) {
	case UpdateSprintWorkItemOwnersVerbAdd:
		for _, owner := range owners {
			if _, ok := ownersMap[owner.UID]; ok {
				return e.ErrUpdateSprintWorkItemOwner.AddErr(errors.New("Owner already exists"))
			}
			workitem.Owners = append(workitem.Owners, owner)
			activityContent += fmt.Sprintf("添加了 %s，", owner.Name)
		}
	case UpdateSprintWorkItemOwnersVerbRemove:
		newOwners := make([]types.UserBriefInfo, 0)
		for _, owner := range workitem.Owners {
			if _, ok := ownersMap[owner.UID]; !ok {
				newOwners = append(newOwners, owner)
			} else {
				activityContent += fmt.Sprintf("移除了 %s，", owner.Name)
			}
		}
		workitem.Owners = newOwners
	default:
		return e.ErrUpdateSprintWorkItemOwner.AddErr(errors.New("Update sprint workitem owners invalid verb"))
	}
	activityContent = strings.TrimRight(activityContent, "，")

	err = mongodb.NewSprintWorkItemCollWithSession(session).UpdateOwners(ctx, id, owners)
	if err != nil {
		return e.ErrUpdateSprintWorkItemOwner.AddErr(errors.Wrapf(err, "Update sprint workitem %s owners", id))
	}

	err = createSprintWorkItemActivity(ctx, session, id, setting.SprintWorkItemActivityTypeEvent, activityContent)
	if err != nil {
		return e.ErrUpdateSprintWorkItemOwner.AddErr(errors.Wrap(err, "Create sprint workitem activity"))
	}

	return nil
}

func createSprintWorkItemActivity(ctx *handler.Context, session mongo.Session, sprintWorkItemID string, activityType setting.SprintWorkItemActivityType, content string) error {

	args := &models.SprintWorkItemActivity{
		SprintWorkItemID: sprintWorkItemID,
		User:             ctx.GenUserBriefInfo(),
		Type:             activityType,
		Content:          content,
	}

	err := mongodb.NewSprintWorkItemActivityCollWithSession(session).Create(ctx, args)
	if err != nil {
		return errors.Wrap(err, "Create sprint workitem activity")
	}

	return nil
}

func ExecSprintWorkItemWorkflow(ctx *handler.Context, workitemIDs []string, workflowName string, workflowArgs *commonmodels.WorkflowV4) error {
	opt := mongodb.ListSprintWorkItemOption{
		IDs: workitemIDs,
	}
	workitem, err := mongodb.NewSprintWorkItemColl().List(ctx, opt)
	if err != nil {
		return e.ErrExecSprintWorkItemTask.AddErr(errors.Wrapf(err, "List workitems by ids %v", workitemIDs))
	}

	sprintID := ""
	stageID := ""
	foundWorkitemIDSet := sets.NewString()
	for _, item := range workitem {
		foundWorkitemIDSet.Insert(item.ID.Hex())

		if sprintID == "" {
			sprintID = item.SprintID
		} else if sprintID != item.SprintID {
			return e.ErrExecSprintWorkItemTask.AddErr(errors.New("Workitems in different sprints"))
		}

		if stageID == "" {
			stageID = item.StageID
		} else if stageID != item.StageID {
			return e.ErrExecSprintWorkItemTask.AddErr(errors.New("Workitems in different stages"))
		}
	}
	for _, id := range workitemIDs {
		if !foundWorkitemIDSet.Has(id) {
			return e.ErrExecSprintWorkItemTask.AddErr(fmt.Errorf("Workitem %s not found", id))
		}
	}

	if sprintID == "" {
		return e.ErrExecSprintWorkItemTask.AddErr(errors.New("Sprint not found"))
	}
	if stageID == "" {
		return e.ErrExecSprintWorkItemTask.AddErr(errors.New("Stage not found"))
	}
	sprint, err := mongodb.NewSprintColl().GetByID(ctx, sprintID)
	if err != nil {
		return e.ErrExecSprintWorkItemTask.AddErr(errors.Wrapf(err, "Get sprint by id %s", sprintID))
	}
	for _, stage := range sprint.Stages {
		if stage.ID == stageID {
			stageWorkItemIDset := sets.NewString(stage.WorkItemIDs...)
			for _, id := range workitemIDs {
				if !stageWorkItemIDset.Has(id) {
					return e.ErrExecSprintWorkItemTask.AddErr(fmt.Errorf("Workitem %s not found in stage", id))
				}
			}
		}
	}

	task, err := workflowservice.CreateWorkflowTaskV4(&workflowservice.CreateWorkflowTaskV4Args{
		Name:    ctx.UserName,
		Account: ctx.Account,
		UserID:  ctx.UserID,
	}, workflowArgs, ctx.Logger)
	if err != nil {
		return e.ErrExecSprintWorkItemTask.AddErr(errors.Wrap(err, "Create workflow task"))
	}

	workitemTask := &commonmodels.SprintWorkItemTask{
		WorkflowName:      workflowName,
		WorkflowTaskID:    task.TaskID,
		SprintWorkItemIDs: workitemIDs,
		Status:            config.StatusCreated,
		CreateTime:        time.Now().Unix(),
		Creator:           ctx.GenUserBriefInfo(),
	}
	err = mongodb.NewSprintWorkItemTaskColl().Create(ctx, workitemTask)
	if err != nil {
		return e.ErrExecSprintWorkItemTask.AddErr(errors.Wrap(err, "Create sprint workitem task"))
	}

	return nil
}

type CloneSprintWorkItemTaskResponse struct {
	Workflow          *commonmodels.WorkflowV4 `json:"workflow"`
	SprintWorkItemIDs []string                 `json:"sprint_workitem_ids"`
}

func CloneSprintWorkItemTask(ctx *handler.Context, workitemTaskID string) (*CloneSprintWorkItemTaskResponse, error) {
	resp := &CloneSprintWorkItemTaskResponse{}
	task, err := mongodb.NewSprintWorkItemTaskColl().GetByID(ctx, workitemTaskID)
	if err != nil {
		return nil, e.ErrCloneSprintWorkItemTask.AddErr(errors.Wrapf(err, "Get workitem task by id %v", workitemTaskID))
	}
	resp.SprintWorkItemIDs = task.SprintWorkItemIDs

	workflow, err := workflowservice.CloneWorkflowTaskV4(task.WorkflowName, task.WorkflowTaskID, ctx.Logger)
	if err != nil {
		return nil, e.ErrCloneSprintWorkItemTask.AddErr(errors.Wrapf(err, "Clone workflow task %s/%d", task.WorkflowName, task.WorkflowTaskID))
	}
	resp.Workflow = workflow

	return resp, nil
}

type ListSprintWorkItemTaskRequset struct {
	WorkflowName string `form:"workflowName" binding:"required"`
	PageNum      int64  `form:"pageNum" binding:"required"`
	PageSize     int64  `form:"pageSize" binding:"required"`
}

type ListSprintWorkItemTaskResponse struct {
	Tasks []*commonmodels.SprintWorkItemTask `json:"tasks"`
	Count int64                              `json:"count"`
}

func ListSprintWorkItemTask(ctx *handler.Context, req *ListSprintWorkItemTaskRequset) (*ListSprintWorkItemTaskResponse, error) {
	tasks, count, err := mongodb.NewSprintWorkItemTaskColl().List(ctx, &mongodb.SprintWorkItemTaskListOption{
		WorkflowName: req.WorkflowName,
		PageNum:      req.PageNum,
		PageSize:     req.PageSize,
	})
	if err != nil {
		return nil, e.ErrListSprintWorkItemTask.AddErr(errors.Wrapf(err, "Get workitem task by workflow name %v", req.WorkflowName))
	}

	resp := &ListSprintWorkItemTaskResponse{
		Tasks: tasks,
		Count: count,
	}

	return resp, nil
}
