package storage

import (
	"github.com/manav03panchal/humantime/internal/model"
)

// GoalRepo provides operations for Goal entities.
type GoalRepo struct {
	db *DB
}

// NewGoalRepo creates a new goal repository.
func NewGoalRepo(db *DB) *GoalRepo {
	return &GoalRepo{db: db}
}

// Create creates a new goal.
func (r *GoalRepo) Create(goal *model.Goal) error {
	goal.Key = model.GenerateGoalKey(goal.ProjectSID)
	return r.db.Set(goal)
}

// Get retrieves a goal by project SID.
func (r *GoalRepo) Get(projectSID string) (*model.Goal, error) {
	goal := &model.Goal{}
	key := model.GenerateGoalKey(projectSID)
	if err := r.db.Get(key, goal); err != nil {
		return nil, err
	}
	return goal, nil
}

// Update updates an existing goal.
func (r *GoalRepo) Update(goal *model.Goal) error {
	return r.db.Set(goal)
}

// Upsert creates or updates a goal.
func (r *GoalRepo) Upsert(goal *model.Goal) error {
	goal.Key = model.GenerateGoalKey(goal.ProjectSID)
	return r.db.Set(goal)
}

// Delete removes a goal by project SID.
func (r *GoalRepo) Delete(projectSID string) error {
	key := model.GenerateGoalKey(projectSID)
	return r.db.Delete(key)
}

// List retrieves all goals.
func (r *GoalRepo) List() ([]*model.Goal, error) {
	return GetAllByPrefix(r.db, model.PrefixGoal+":", func() *model.Goal {
		return &model.Goal{}
	})
}

// Exists checks if a goal exists for the given project.
func (r *GoalRepo) Exists(projectSID string) (bool, error) {
	key := model.GenerateGoalKey(projectSID)
	return r.db.Exists(key)
}
