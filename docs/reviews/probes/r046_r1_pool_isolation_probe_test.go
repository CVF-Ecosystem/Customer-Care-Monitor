package engine

import (
 "testing"
 "github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
)

// Reviewer-only observation: never repairs the maintained worker helper.
func TestR046R1BoundaryPoolRestoresOriginalStatement(t *testing.T) {
 setupFLFixture(t)
 originalDB := db.DB
 originalStatement := originalDB.Statement
 originalPool := originalStatement.ConnPool
 // Ensure this diagnostic itself restores both fields after the child cleanup.
 defer func() { originalStatement.ConnPool = originalPool; db.DB = originalDB }()
 t.Run("install_and_cleanup", func(t *testing.T) {
  installBoundaryPool(t, &flBoundaryPool{})
  if originalStatement.ConnPool != originalPool {
   t.Error("worker helper mutated original Statement.ConnPool before cleanup")
  }
 })
 if db.DB != originalDB { t.Error("worker cleanup did not restore DB pointer") }
 if originalDB.Statement.ConnPool != originalPool {
  t.Fatal("worker cleanup restored DB pointer but retained injected pool in original Statement")
 }
}
