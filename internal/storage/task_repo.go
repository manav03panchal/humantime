package storage

import (
	"github.com/manav03panchal/humantime/internal/model"
)

// TaskRepo provides operations for Task entities.
type TaskRepo struct {
	db *DB
}

// NewTaskRepo creates a new task repository.
func NewTaskRepo(db *DB) *TaskRepo {
	return &TaskRepo{db: db}
}

// Create creates a new task.
func (r *TaskRepo) Create(task *model.Task) error {
	task.Key = model.GenerateTaskKey(task.ProjectSID, task.SID)
	return r.db.Set(task)
}

// Get retrieves a task by project SID and task SID.
func (r *TaskRepo) Get(projectSID, taskSID string) (*model.Task, error) {
	task := &model.Task{}
	key := model.GenerateTaskKey(projectSID, taskSID)
	if err := r.db.Get(key, task); err != nil {
		return nil, err
	}
	return task, nil
}

// GetOrCreate retrieves a task, creating it if it doesn't exist.
// This operation is atomic to prevent race conditions.
func (r *TaskRepo) GetOrCreate(projectSID, taskSID, displayName string) (*model.Task, bool, error) {
	key := model.GenerateTaskKey(projectSID, taskSID)
	existing := &model.Task{}

	result, created, err := r.db.GetOrCreate(key, existing, func() model.Model {
		return model.NewTask(projectSID, taskSID, displayName, "")
	})
	if err != nil {
		return nil, false, err
	}

	return result.(*model.Task), created, nil
}

// Update updates an existing task.
func (r *TaskRepo) Update(task *model.Task) error {
	return r.db.Set(task)
}

// Delete removes a task.
func (r *TaskRepo) Delete(projectSID, taskSID string) error {
	key := model.GenerateTaskKey(projectSID, taskSID)
	return r.db.Delete(key)
}

// List retrieves all tasks.
func (r *TaskRepo) List() ([]*model.Task, error) {
	return GetAllByPrefix(r.db, model.PrefixTask+":", func() *model.Task {
		return &model.Task{}
	})
}

// ListByProject retrieves all tasks for a specific project.
func (r *TaskRepo) ListByProject(projectSID string) ([]*model.Task, error) {
	prefix := model.PrefixTask + ":" + projectSID + ":"
	return GetAllByPrefix(r.db, prefix, func() *model.Task {
		return &model.Task{}
	})
}

// Exists checks if a task exists.
func (r *TaskRepo) Exists(projectSID, taskSID string) (bool, error) {
	key := model.GenerateTaskKey(projectSID, taskSID)
	return r.db.Exists(key)
}
