package db

import (
	"fmt"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

// EnsureSingleWorkspace fails closed on a database from the multi-company app.
// Such data needs an explicit migration decision; choosing one row would hide data.
func EnsureSingleWorkspace() error {
	var count int64
	if err := DB.Model(&models.Tenant{}).Count(&count).Error; err != nil {
		return fmt.Errorf("count workspaces: %w", err)
	}
	if count > 1 {
		return fmt.Errorf("single-workspace mode requires at most one workspace; found %d", count)
	}
	return nil
}
