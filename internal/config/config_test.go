package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rylero/robot-creator/internal/config"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	data := "team: 6328\npackage: frc.robot\nsubsystems:\n  - name: Shooter\n    type: flywheel\n"
	os.WriteFile(filepath.Join(dir, "robot-creator.yaml"), []byte(data), 0644)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Team != 6328 {
		t.Errorf("Team = %d, want 6328", cfg.Team)
	}
	if cfg.Package != "frc.robot" {
		t.Errorf("Package = %q, want frc.robot", cfg.Package)
	}
	if len(cfg.Subsystems) != 1 || cfg.Subsystems[0].Name != "Shooter" {
		t.Errorf("Subsystems = %v, want [{Shooter flywheel}]", cfg.Subsystems)
	}
}

func TestLoad_Missing(t *testing.T) {
	_, err := config.Load(t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing config")
	}
}

func TestSave(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{Team: 254, Package: "frc.robot"}
	if err := config.Save(dir, cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	loaded, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load after Save error: %v", err)
	}
	if loaded.Team != 254 {
		t.Errorf("Team = %d, want 254", loaded.Team)
	}
}

func TestHasSubsystem(t *testing.T) {
	cfg := &config.Config{
		Subsystems: []config.Subsystem{{Name: "Shooter", Type: "flywheel"}},
	}
	if !cfg.HasSubsystem("Shooter") {
		t.Error("HasSubsystem(Shooter) = false, want true")
	}
	if cfg.HasSubsystem("Hopper") {
		t.Error("HasSubsystem(Hopper) = true, want false")
	}
}
