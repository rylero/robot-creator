package injector_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rylero/robot-creator/internal/injector"
)

func loadFixture(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("testdata/RobotContainer.java")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "RobotContainer.java")
	os.WriteFile(path, data, 0644)
	return path
}

func TestInject_AddsImports(t *testing.T) {
	path := loadFixture(t)
	inj := injector.New(path)
	err := inj.Inject(injector.SubsystemInjection{
		Name: "Shooter", NameLower: "shooter", Package: "frc.robot",
	})
	if err != nil {
		t.Fatalf("Inject() error: %v", err)
	}
	content, _ := os.ReadFile(path)
	s := string(content)
	if !strings.Contains(s, "import frc.robot.subsystems.shooter.Shooter;") {
		t.Error("missing Shooter import")
	}
	if !strings.Contains(s, "import frc.robot.subsystems.shooter.ShooterIO;") {
		t.Error("missing ShooterIO import")
	}
	if !strings.Contains(s, "import frc.robot.subsystems.shooter.ShooterIOTalonFX;") {
		t.Error("missing ShooterIOTalonFX import")
	}
	if !strings.Contains(s, "import frc.robot.subsystems.shooter.ShooterIOSim;") {
		t.Error("missing ShooterIOSim import")
	}
}

func TestInject_AddsFieldDeclaration(t *testing.T) {
	path := loadFixture(t)
	inj := injector.New(path)
	inj.Inject(injector.SubsystemInjection{Name: "Shooter", NameLower: "shooter", Package: "frc.robot"})
	content, _ := os.ReadFile(path)
	if !strings.Contains(string(content), "private final Shooter shooter;") {
		t.Error("missing field declaration")
	}
}

func TestInject_AddsSwitchCases(t *testing.T) {
	path := loadFixture(t)
	inj := injector.New(path)
	inj.Inject(injector.SubsystemInjection{Name: "Shooter", NameLower: "shooter", Package: "frc.robot"})
	content, _ := os.ReadFile(path)
	s := string(content)
	if !strings.Contains(s, "shooter = new Shooter(new ShooterIOTalonFX());") {
		t.Error("missing REAL case instantiation")
	}
	if !strings.Contains(s, "shooter = new Shooter(new ShooterIOSim());") {
		t.Error("missing SIM case instantiation")
	}
	if !strings.Contains(s, "shooter = new Shooter(new ShooterIO() {});") {
		t.Error("missing default case instantiation")
	}
}

func TestInject_BreakStatementsPreserved(t *testing.T) {
	path := loadFixture(t)
	inj := injector.New(path)
	inj.Inject(injector.SubsystemInjection{Name: "Shooter", NameLower: "shooter", Package: "frc.robot"})
	content, _ := os.ReadFile(path)
	count := strings.Count(string(content), "break;")
	if count != 3 {
		t.Errorf("break count = %d, want 3", count)
	}
}

func TestInject_NoSwitchBlock_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "RobotContainer.java")
	os.WriteFile(path, []byte("public class RobotContainer { private final Drive drive; }"), 0644)
	inj := injector.New(path)
	err := inj.Inject(injector.SubsystemInjection{Name: "Shooter", NameLower: "shooter", Package: "frc.robot"})
	if err == nil {
		t.Error("expected error for missing switch block")
	}
}

func TestEject_RoundTrip(t *testing.T) {
	original, err := os.ReadFile("testdata/RobotContainer.java")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	path := loadFixture(t)
	inj := injector.New(path)
	si := injector.SubsystemInjection{Name: "Shooter", NameLower: "shooter", Package: "frc.robot"}

	if err := inj.Inject(si); err != nil {
		t.Fatalf("Inject() error: %v", err)
	}
	if err := inj.Eject(si); err != nil {
		t.Fatalf("Eject() error: %v", err)
	}

	result, _ := os.ReadFile(path)
	if string(result) != string(original) {
		t.Error("file after Inject+Eject does not match original")
	}
}

func TestEject_RemovesLines(t *testing.T) {
	path := loadFixture(t)
	inj := injector.New(path)
	si := injector.SubsystemInjection{Name: "Shooter", NameLower: "shooter", Package: "frc.robot"}
	inj.Inject(si)
	inj.Eject(si)

	content, _ := os.ReadFile(path)
	s := string(content)
	for _, banned := range []string{
		"import frc.robot.subsystems.shooter.Shooter;",
		"private final Shooter shooter;",
		"shooter = new Shooter(new ShooterIOTalonFX());",
		"shooter = new Shooter(new ShooterIOSim());",
		"shooter = new Shooter(new ShooterIO() {});",
	} {
		if strings.Contains(s, banned) {
			t.Errorf("Eject left line: %s", banned)
		}
	}
}

func TestEject_NotFound_ReturnsError(t *testing.T) {
	path := loadFixture(t)
	inj := injector.New(path)
	err := inj.Eject(injector.SubsystemInjection{Name: "Nonexistent", NameLower: "nonexistent", Package: "frc.robot"})
	if err == nil {
		t.Error("expected error when ejecting subsystem not present in file")
	}
}
