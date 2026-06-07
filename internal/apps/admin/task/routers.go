/*
Copyright 2025 linux.do

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

package task

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/linux-do/credit/internal/task"
	"github.com/linux-do/credit/internal/task/scheduler"
	"github.com/linux-do/credit/internal/util"
)

// ListTaskTypes 获取支持的任务类型列表
// @Summary 获取支持的任务类型
// @Description 返回系统支持的所有可调度任务类型列表，需要管理员权限
// @Tags admin
// @Produce json
// @Success 200 {object} util.ResponseAny
// @Router /api/v1/admin/tasks/types [get]
func ListTaskTypes(c *gin.Context) {
	c.JSON(http.StatusOK, util.OK(task.DispatchableTasks))
}

// DispatchTaskRequest 下发任务请求
type DispatchTaskRequest struct {
	TaskType  string     `json:"task_type" binding:"required"`
	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
	UserID    *uint64    `json:"user_id"`
}

// DispatchTask 下发任务
// @Summary 下发异步任务
// @Description 手动触发指定类型的异步任务，支持指定时间范围和用户，需要管理员权限
// @Tags admin
// @Accept json
// @Produce json
// @Param request body DispatchTaskRequest true "任务请求参数"
// @Success 200 {object} util.ResponseAny
// @Router /api/v1/admin/tasks/dispatch [post]
func DispatchTask(c *gin.Context) {
	var req DispatchTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	meta := task.GetTaskMeta(req.TaskType)
	if meta == nil {
		c.JSON(http.StatusBadRequest, util.Err(InvalidTaskType))
		return
	}

	var taskInfo *asynq.Task
	var taskID string

	taskInfo = asynq.NewTask(meta.AsynqTask, nil)
	taskID = fmt.Sprintf("manual_%s", req.TaskType)

	_, err := scheduler.AsynqClient.Enqueue(
		taskInfo,
		asynq.TaskID(taskID),
		asynq.MaxRetry(meta.MaxRetry),
		asynq.Queue(meta.Queue),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(fmt.Sprintf("%s: %v", TaskDispatchFailed, err)))
		return
	}

	c.JSON(http.StatusOK, util.OKNil())
}
