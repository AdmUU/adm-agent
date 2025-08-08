// Copyright 2024-2025 Admin.IM <dev@admin.im>
// SPDX-License-Identifier: GPL-3.0-or-later

package components

import (
	"sync"
)

type TaskHandler interface {
	ValidateData(data map[string]interface{}) error
	PreProcess(data map[string]interface{}) (processedData map[string]interface{}, response map[string]interface{}, err error)
	Execute(data map[string]interface{}, taskId string, stopChan <-chan struct{}, responseSender ResponseSender) error
	GetTaskType() string
}

// ResponseSender interface for sending responses
type ResponseSender interface {
	SendMessage(event string, data map[string]interface{}) error
}

// TaskRegistry manages registration and retrieval of task handlers
type TaskRegistry struct {
	handlers map[string]TaskHandler
	mutex    sync.RWMutex
}

// NewTaskRegistry creates a new task registry instance
func NewTaskRegistry() *TaskRegistry {
	return &TaskRegistry{
		handlers: make(map[string]TaskHandler),
	}
}

// RegisterHandler registers a task handler for a specific task type
func (tr *TaskRegistry) RegisterHandler(handler TaskHandler) {
	tr.mutex.Lock()
	defer tr.mutex.Unlock()
	tr.handlers[handler.GetTaskType()] = handler
}

// GetHandler retrieves a task handler by task type
func (tr *TaskRegistry) GetHandler(taskType string) (TaskHandler, bool) {
	tr.mutex.RLock()
	defer tr.mutex.RUnlock()
	handler, exists := tr.handlers[taskType]
	return handler, exists
}

// GetAllHandlers returns all registered task types
func (tr *TaskRegistry) GetAllHandlers() []string {
	tr.mutex.RLock()
	defer tr.mutex.RUnlock()
	var types []string
	for taskType := range tr.handlers {
		types = append(types, taskType)
	}
	return types
}