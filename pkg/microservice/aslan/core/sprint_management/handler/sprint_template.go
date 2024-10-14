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

package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/models"
	commonutil "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/util"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/sprint_management/service"
	"github.com/koderover/zadig/v2/pkg/setting"
	internalhandler "github.com/koderover/zadig/v2/pkg/shared/handler"
	e "github.com/koderover/zadig/v2/pkg/tool/errors"
)

// @Summary Get Sprint Template
// @Description Get Sprint Template
// @Tags 	SprintManagement
// @Accept 	json
// @Produce json
// @Param 	projectName		query		string							true	"project name"
// @Param 	id				path		string							true	"sprint template id"
// @Success 200 			{object} 	models.SprintTemplate
// @Router /api/aslan/sprint_management/v1/sprint_template/{id} [get]
func GetSprintTemplate(c *gin.Context) {
	ctx, err := internalhandler.NewContextWithAuthorization(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	if err != nil {
		ctx.RespErr = fmt.Errorf("authorization Info Generation failed: err %s", err)
		ctx.UnAuthorized = true
		return
	}

	projectName := c.Query("projectName")
	if !ctx.Resources.IsSystemAdmin {
		if _, ok := ctx.Resources.ProjectAuthInfo[projectName]; !ok {
			ctx.UnAuthorized = true
			return
		}
		if !ctx.Resources.ProjectAuthInfo[projectName].IsProjectAdmin &&
			!ctx.Resources.ProjectAuthInfo[projectName].SprintTemplateManagement.Edit {
			ctx.UnAuthorized = true
			return
		}
	}

	err = commonutil.CheckZadigEnterpriseLicense()
	if err != nil {
		ctx.RespErr = err
		return
	}

	ctx.Resp, ctx.RespErr = service.GetSprintTemplate(ctx, c.Param("id"))
}

// @Summary Create Sprint Template
// @Description Create Sprint Template
// @Tags 	SprintManagement
// @Accept 	json
// @Produce json
// @Param 	projectName		query		string							true	"project name"
// @Param 	body 			body 		models.SprintTemplate 			true 	"body"
// @Success 200
// @Router /api/aslan/sprint_management/v1/sprint_template [post]
func CreateSprintTemplate(c *gin.Context) {
	ctx, err := internalhandler.NewContextWithAuthorization(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	if err != nil {
		ctx.RespErr = fmt.Errorf("authorization Info Generation failed: err %s", err)
		ctx.UnAuthorized = true
		return
	}

	projectName := c.Query("projectName")
	if !ctx.Resources.IsSystemAdmin {
		if _, ok := ctx.Resources.ProjectAuthInfo[projectName]; !ok {
			ctx.UnAuthorized = true
			return
		}
		if !ctx.Resources.ProjectAuthInfo[projectName].IsProjectAdmin &&
			!ctx.Resources.ProjectAuthInfo[projectName].SprintTemplateManagement.Edit {
			ctx.UnAuthorized = true
			return
		}
	}

	req := new(models.SprintTemplate)
	if err := c.ShouldBindJSON(req); err != nil {
		ctx.RespErr = e.ErrInvalidParam.AddDesc(err.Error())
		return
	}

	err = commonutil.CheckZadigEnterpriseLicense()
	if err != nil {
		ctx.RespErr = err
		return
	}

	ctx.RespErr = service.CreateSprintTemplate(ctx, req)
}

// @Summary Update Sprint Template
// @Description Update Sprint Template
// @Tags 	SprintManagement
// @Accept 	json
// @Produce json
// @Param 	projectName		query		string							true	"project name"
// @Param 	id				path		string							true	"sprint template id"
// @Param 	body 			body 		models.SprintTemplate 			true 	"body"
// @Success 200
// @Router /api/aslan/sprint_management/v1/sprint_template/{id} [put]
func UpdateSprintTemplate(c *gin.Context) {
	ctx, err := internalhandler.NewContextWithAuthorization(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	if err != nil {
		ctx.RespErr = fmt.Errorf("authorization Info Generation failed: err %s", err)
		ctx.UnAuthorized = true
		return
	}

	projectName := c.Query("projectName")
	if !ctx.Resources.IsSystemAdmin {
		if _, ok := ctx.Resources.ProjectAuthInfo[projectName]; !ok {
			ctx.UnAuthorized = true
			return
		}
		if !ctx.Resources.ProjectAuthInfo[projectName].IsProjectAdmin &&
			!ctx.Resources.ProjectAuthInfo[projectName].SprintTemplateManagement.Edit {
			ctx.UnAuthorized = true
			return
		}
	}

	err = commonutil.CheckZadigEnterpriseLicense()
	if err != nil {
		ctx.RespErr = err
		return
	}

	req := new(models.SprintTemplate)
	if err := c.ShouldBindJSON(req); err != nil {
		ctx.RespErr = e.ErrInvalidParam.AddDesc(err.Error())
		return
	}

	data, err := c.Copy().GetRawData()
	if err != nil {
		ctx.RespErr = e.ErrInvalidParam.AddErr(err)
		return
	}
	internalhandler.InsertDetailedOperationLog(c, ctx.UserName, projectName, setting.OperationSceneSprintManagement, "更新", "流程", req.Name, string(data), ctx.Logger, "")

	ctx.RespErr = service.UpdateSprintTemplate(ctx, c.Param("id"), req)
}

// @Summary Delete Sprint Template
// @Description Delete Sprint Template
// @Tags 	SprintManagement
// @Accept 	json
// @Produce json
// @Param 	projectName		query		string							true	"project name"
// @Param 	id				path		string							true	"sprint template id"
// @Success 200
// @Router /api/aslan/sprint_management/v1/sprint_template/{id} [delete]
func DeleteSprintTemplate(c *gin.Context) {
	ctx, err := internalhandler.NewContextWithAuthorization(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	if err != nil {
		ctx.RespErr = fmt.Errorf("authorization Info Generation failed: err %s", err)
		ctx.UnAuthorized = true
		return
	}

	projectName := c.Query("projectName")
	if !ctx.Resources.IsSystemAdmin {
		if _, ok := ctx.Resources.ProjectAuthInfo[projectName]; !ok {
			ctx.UnAuthorized = true
			return
		}
		if !ctx.Resources.ProjectAuthInfo[projectName].IsProjectAdmin &&
			!ctx.Resources.ProjectAuthInfo[projectName].SprintTemplateManagement.Edit {
			ctx.UnAuthorized = true
			return
		}
	}

	err = commonutil.CheckZadigEnterpriseLicense()
	if err != nil {
		ctx.RespErr = err
		return
	}

	ctx.RespErr = service.DeleteSprintTemplate(ctx, ctx.UserName, c.Param("id"))
}

// @Summary List Sprint Templates
// @Description List Sprint Templates
// @Tags 	SprintManagement
// @Accept 	json
// @Produce json
// @Param 	projectName		query		string							true	"project name"
// @Param 	pageNum 		query		int								true	"page num"
// @Param 	pageSize 		query		int								true	"page size"
// @Success 200 			{object} 	service.ListSprintTemplateResp
// @Router /api/aslan/sprint_management/v1/sprint_template [get]
func ListSprintTemplates(c *gin.Context) {
	ctx, err := internalhandler.NewContextWithAuthorization(c)
	defer func() { internalhandler.JSONResponse(c, ctx) }()

	if err != nil {
		ctx.RespErr = fmt.Errorf("authorization Info Generation failed: err %s", err)
		ctx.UnAuthorized = true
		return
	}

	projectName := c.Query("projectName")
	if !ctx.Resources.IsSystemAdmin {
		if _, ok := ctx.Resources.ProjectAuthInfo[projectName]; !ok {
			ctx.UnAuthorized = true
			return
		}
		if !ctx.Resources.ProjectAuthInfo[projectName].IsProjectAdmin &&
			!ctx.Resources.ProjectAuthInfo[projectName].SprintTemplateManagement.Edit {
			ctx.UnAuthorized = true
			return
		}
	}

	err = commonutil.CheckZadigEnterpriseLicense()
	if err != nil {
		ctx.RespErr = err
		return
	}

	opt := new(service.ListSprintTemplateOption)
	if err := c.ShouldBindQuery(&opt); err != nil {
		ctx.RespErr = e.ErrInvalidParam.AddDesc(err.Error())
		return
	}

	ctx.Resp, ctx.RespErr = service.ListSprintTemplate(ctx, opt)
}
