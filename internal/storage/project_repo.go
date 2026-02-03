package storage

import (
	"github.com/manav03panchal/humantime/internal/model"
)

// ProjectRepo provides operations for Project entities.
type ProjectRepo struct {
	db *DB
}

// NewProjectRepo creates a new project repository.
func NewProjectRepo(db *DB) *ProjectRepo {
	return &ProjectRepo{db: db}
}

// Create creates a new project.
func (r *ProjectRepo) Create(project *model.Project) error {
	project.Key = model.GenerateProjectKey(project.SID)
	return r.db.Set(project)
}

// Get retrieves a project by SID.
func (r *ProjectRepo) Get(sid string) (*model.Project, error) {
	project := &model.Project{}
	key := model.GenerateProjectKey(sid)
	if err := r.db.Get(key, project); err != nil {
		return nil, err
	}
	return project, nil
}

// GetOrCreate retrieves a project by SID, creating it if it doesn't exist.
// This operation is atomic to prevent race conditions.
func (r *ProjectRepo) GetOrCreate(sid, displayName string) (*model.Project, bool, error) {
	key := model.GenerateProjectKey(sid)
	existing := &model.Project{}

	result, created, err := r.db.GetOrCreate(key, existing, func() model.Model {
		return model.NewProject(sid, displayName, "")
	})
	if err != nil {
		return nil, false, err
	}

	return result.(*model.Project), created, nil
}

// Update updates an existing project.
func (r *ProjectRepo) Update(project *model.Project) error {
	return r.db.Set(project)
}

// Delete removes a project by SID.
func (r *ProjectRepo) Delete(sid string) error {
	key := model.GenerateProjectKey(sid)
	return r.db.Delete(key)
}

// List retrieves all projects.
func (r *ProjectRepo) List() ([]*model.Project, error) {
	return GetAllByPrefix(r.db, model.PrefixProject+":", func() *model.Project {
		return &model.Project{}
	})
}

// Exists checks if a project exists by SID.
func (r *ProjectRepo) Exists(sid string) (bool, error) {
	key := model.GenerateProjectKey(sid)
	return r.db.Exists(key)
}
