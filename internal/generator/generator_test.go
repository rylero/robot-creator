package generator_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ced4rtree/robot-creator/internal/generator"
	tmpl "github.com/ced4rtree/robot-creator/internal/template"
)

func TestGenerateSubsystem_Generic(t *testing.T) {
	dir := t.TempDir()
	ctx := generator.SubsystemContext{
		Name:       "Shooter",
		NameLower:  "shooter",
		Type:       "generic",
		Package:    "frc.robot",
		TeamNumber: 6328,
	}

	gen := generator.New(tmpl.NewEmbeddedSource())
	if err := gen.GenerateSubsystem(ctx, dir); err != nil {
		t.Fatalf("GenerateSubsystem() error: %v", err)
	}

	outDir := filepath.Join(dir, "src", "main", "java", "frc", "robot", "subsystems", "shooter")
	expectedFiles := []string{
		"Shooter.java", "ShooterIO.java", "ShooterIOTalonFX.java",
		"ShooterIOSim.java", "ShooterConstants.java",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(outDir, f)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s not found", f)
		}
	}

	content, _ := os.ReadFile(filepath.Join(outDir, "Shooter.java"))
	s := string(content)
	if !strings.Contains(s, "public class Shooter extends SubsystemBase") {
		t.Error("Shooter.java missing class declaration")
	}
	if strings.Contains(s, "{{") {
		t.Error("Shooter.java contains unrendered template syntax")
	}
	if !strings.Contains(s, "package frc.robot.subsystems.shooter") {
		t.Error("Shooter.java missing package declaration")
	}
}

func TestGenerateSubsystem_Flywheel(t *testing.T) {
	dir := t.TempDir()
	ctx := generator.SubsystemContext{
		Name: "Flywheel", NameLower: "flywheel",
		Type: "flywheel", Package: "frc.robot",
	}
	gen := generator.New(tmpl.NewEmbeddedSource())
	if err := gen.GenerateSubsystem(ctx, dir); err != nil {
		t.Fatalf("GenerateSubsystem() error: %v", err)
	}
	outDir := filepath.Join(dir, "src", "main", "java", "frc", "robot", "subsystems", "flywheel")
	content, _ := os.ReadFile(filepath.Join(outDir, "Flywheel.java"))
	if !strings.Contains(string(content), "atTargetVelocity") {
		t.Error("flywheel Subsystem.java missing atTargetVelocity")
	}
}

func TestGenerateSubsystem_DuplicateDir(t *testing.T) {
	dir := t.TempDir()
	ctx := generator.SubsystemContext{
		Name: "Test", NameLower: "test", Type: "generic", Package: "frc.robot",
	}
	gen := generator.New(tmpl.NewEmbeddedSource())
	gen.GenerateSubsystem(ctx, dir)
	if err := gen.GenerateSubsystem(ctx, dir); err != nil {
		t.Fatalf("second GenerateSubsystem() error: %v", err)
	}
}
