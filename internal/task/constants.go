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

const (
	CleanupUnusedUploadsTask = "upload:cleanup_unused"
)

const (
	QueueDefault = "default"
)

// 管理员可下发的任务类型标识
const (
	TaskTypeCleanupUploads = "cleanup_unused_uploads"
)

// TaskMeta 任务元数据
type TaskMeta struct {
	Type         string
	AsynqTask    string
	Name         string
	Description  string
	SupportsTime bool
	MaxRetry     int
	Queue        string
}

// DispatchableTasks 可下发的任务列表
var DispatchableTasks = []TaskMeta{
	{
		Type:         TaskTypeCleanupUploads,
		AsynqTask:    CleanupUnusedUploadsTask,
		Name:         "清理未使用上传",
		Description:  "清理超过1小时未使用的上传文件",
		SupportsTime: false,
		MaxRetry:     3,
		Queue:        QueueDefault,
	},
}

// GetTaskMeta 根据任务类型获取元数据
func GetTaskMeta(taskType string) *TaskMeta {
	for _, t := range DispatchableTasks {
		if t.Type == taskType {
			return &t
		}
	}
	return nil
}
