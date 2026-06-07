/*
Copyright 2026 linux.do

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
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/linux-do/credit/internal/apps/oauth"
	"github.com/linux-do/credit/internal/model"
	"github.com/linux-do/credit/internal/task"
	"github.com/linux-do/credit/internal/testhelper"
	"github.com/linux-do/credit/internal/util"
)

func setupTestRouter(authUser *model.User) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	adminGroup := r.Group("/api/v1/admin")

	// Mock authentication middleware
	adminGroup.Use(func(c *gin.Context) {
		if authUser != nil {
			util.SetToContext(c, oauth.UserObjKey, authUser)
		}
		c.Next()
	})

	adminGroup.GET("/tasks/types", ListTaskTypes)
	adminGroup.POST("/tasks/dispatch", DispatchTask)
	return r
}

func TestListTaskTypes(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	adminUser := &model.User{ID: 1001, Username: "admin", IsAdmin: true, SignKey: "admin_key"}
	router := setupTestRouter(adminUser)

	req, _ := http.NewRequest("GET", "/api/v1/admin/tasks/types", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}

	var resp util.ResponseAny
	json.Unmarshal(w.Body.Bytes(), &resp)

	dataBytes, _ := json.Marshal(resp.Data)
	var taskMetas []task.TaskMeta
	json.Unmarshal(dataBytes, &taskMetas)

	if len(taskMetas) == 0 {
		t.Error("expected at least one dispatchable task type")
	}

	foundCleanup := false
	for _, m := range taskMetas {
		if m.Type == task.TaskTypeCleanupUploads {
			foundCleanup = true
			break
		}
	}
	if !foundCleanup {
		t.Errorf("expected task type %s to be listed", task.TaskTypeCleanupUploads)
	}
}

func TestDispatchTask(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	adminUser := &model.User{ID: 1001, Username: "admin", IsAdmin: true, SignKey: "admin_key"}
	router := setupTestRouter(adminUser)

	t.Run("dispatch valid task successfully", func(t *testing.T) {
		payload := DispatchTaskRequest{
			TaskType: task.TaskTypeCleanupUploads,
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/v1/admin/tasks/dispatch", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("dispatch invalid task type failure", func(t *testing.T) {
		payload := DispatchTaskRequest{
			TaskType: "invalid_task_type",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/v1/admin/tasks/dispatch", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}

		var resp util.ResponseAny
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.ErrorMsg != InvalidTaskType {
			t.Errorf("expected error message '%s', got '%s'", InvalidTaskType, resp.ErrorMsg)
		}
	})
}
