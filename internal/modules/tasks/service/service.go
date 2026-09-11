package service

import (
	"context"
	"encoding/json"
	"strings"
	"taskmanagement/internal/application/errorcode"
	"taskmanagement/internal/application/notification"
	"taskmanagement/internal/modules/tasks/domain"
	"taskmanagement/internal/platform/id"
	"time"
)

type taskService struct {
	Repository          domain.Repository
	NotificationService notification.NotificationService
}

func NewTaskService(taskRepository domain.Repository, notificationService notification.NotificationService) domain.Service {
	return &taskService{
		Repository:          taskRepository,
		NotificationService: notificationService,
	}
}

func (taskService *taskService) Create(ctx context.Context, input domain.CreateInput) (domain.CreateOutput, error) {
	userID, idempotencyKey, task := input.UserID, input.IdempotencyKey, input.Task
	task.TeamID = strings.TrimSpace(task.TeamID)
	task.Title = strings.TrimSpace(task.Title)
	task.Description = strings.TrimSpace(task.Description)
	task.Status = strings.TrimSpace(task.Status)
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	task.CreatedAt = now
	task.UpdatedAt = now
	isMember, err := taskService.Repository.Member(ctx, task.TeamID, userID)
	if err != nil {
		return domain.CreateOutput{}, err
	}

	if !isMember {
		return domain.CreateOutput{}, domain.ErrForbidden
	}

	task.CreatorID = userID
	task.ID = id.New()
	responseBody, err := json.Marshal(task)
	if err != nil {
		return domain.CreateOutput{}, err
	}
	return taskService.Repository.CreateIdempotent(ctx, task, userID, idempotencyKey, responseBody)
}

func (taskService *taskService) List(ctx context.Context, input domain.ListInput) ([]domain.Task, int, error) {
	return taskService.Repository.List(ctx, input.UserID, input.TeamID, input.Status, input.Search, input.Page, input.Limit)
}

func (taskService *taskService) Get(ctx context.Context, input domain.GetInput) (domain.Task, error) {
	return taskService.Repository.Get(ctx, input.TaskID, input.UserID)
}

func (taskService *taskService) Update(ctx context.Context, input domain.UpdateInput) (domain.Task, error) {
	taskID, userID, task := input.TaskID, input.UserID, input.Task
	existingTask, err := taskService.Repository.Get(ctx, taskID, userID)
	if err != nil {
		return existingTask, err
	}

	if existingTask.CreatorID != userID && (task.Title != "" && task.Title != existingTask.Title || task.Description != "" && task.Description != existingTask.Description) {
		return existingTask, domain.ErrForbidden
	}

	if task.Status == "" {
		task.Status = existingTask.Status
	}

	if task.Title == "" {
		task.Title = existingTask.Title
	}

	if task.Description == "" {
		task.Description = existingTask.Description
	}
	task.ID = taskID
	task.TeamID = existingTask.TeamID
	task.CreatorID = existingTask.CreatorID
	if err = taskService.Repository.Update(ctx, task, userID); err != nil {
		return existingTask, err
	}
	updatedTask, err := taskService.Repository.Get(ctx, taskID, userID)
	if err != nil {
		return existingTask, err
	}
	return updatedTask, nil
}

func (taskService *taskService) Delete(ctx context.Context, input domain.DeleteInput) error {
	return taskService.Repository.Delete(ctx, input.TaskID, input.UserID)
}

func (taskService *taskService) Assign(ctx context.Context, input domain.AssignInput) (domain.Task, error) {
	taskID, userID, targetUserID := input.TaskID, input.UserID, input.TargetUserID
	task, err := taskService.Repository.Get(ctx, taskID, userID)
	if err != nil || task.CreatorID != userID {
		return task, domain.ErrNotFound
	}

	if targetUserID != "" {
		isMember, err := taskService.Repository.Member(ctx, task.TeamID, targetUserID)
		if err != nil {
			return task, err
		}

		if !isMember {
			return task, domain.ErrForbidden
		}
	}
	action := errorcode.TaskAssigned
	if targetUserID == "" {
		action = errorcode.TaskUnassigned
	}

	var assignee *string
	if targetUserID != "" {
		assignee = &targetUserID
	}

	var notify func() error
	notifyUser := targetUserID
	if notifyUser == "" && task.AssigneeID != nil {
		notifyUser = *task.AssigneeID
	}
	if notifyUser != "" && taskService.NotificationService != nil {
		notify = func() error {
			message := "task assigned"
			if targetUserID == "" {
				message = "task unassigned"
			}
			return taskService.NotificationService.Send(ctx, notification.Notification{
				UserID:  notifyUser,
				TaskID:  taskID,
				Message: message,
			})
		}
	}

	if err = taskService.Repository.Assign(ctx, taskID, userID, assignee, action, notify); err != nil {
		return task, err
	}
	if targetUserID == "" {
		task.AssigneeID = nil
	} else {
		task.AssigneeID = &targetUserID
	}

	return task, nil
}
