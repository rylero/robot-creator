package injector_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rylero/robot-creator/v3/internal/injector"
)

// The fixtures are RobotContainer.java and Robot.java from 6238/2026RobotSystemcore.
func loadFixtures(t *testing.T) *injector.Injector {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"RobotContainer.java", "Robot.java"} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatalf("reading fixture: %v", err)
		}
		os.WriteFile(filepath.Join(dir, name), data, 0644)
	}
	return injector.New(filepath.Join(dir, "RobotContainer.java"), filepath.Join(dir, "Robot.java"))
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

var climber = injector.SubsystemInjection{Name: "Climber", NameLower: "climber", Package: "frc.robot"}

func TestInject_AddsImports(t *testing.T) {
	inj := loadFixtures(t)
	if err := inj.Inject(climber); err != nil {
		t.Fatalf("Inject() error: %v", err)
	}
	s := read(t, inj.RobotContainerPath)
	for _, want := range []string{
		"import frc.robot.subsystems.climber.Climber;",
		"import frc.robot.subsystems.climber.ClimberIO;",
		"import frc.robot.subsystems.climber.ClimberIOTalonFX;",
		"import frc.robot.subsystems.climber.ClimberIOSim;",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestInject_AddsPublicFieldAfterLastField(t *testing.T) {
	inj := loadFixtures(t)
	inj.Inject(climber)
	s := read(t, inj.RobotContainerPath)
	want := "  public final Superstructure superstructure;\n  public final Climber climber;\n"
	if !strings.Contains(s, want) {
		t.Errorf("field not placed after the last final field:\n%s", s[:1500])
	}
}

func TestInject_AddsSwitchCases(t *testing.T) {
	inj := loadFixtures(t)
	inj.Inject(climber)
	s := read(t, inj.RobotContainerPath)
	for _, want := range []string{
		"        intake = new Intake(new IntakeIOTalonFX());\n        climber = new Climber(new ClimberIOTalonFX());\n        break;",
		"        intake = new Intake(new IntakeIOSim());\n        climber = new Climber(new ClimberIOSim());\n        break;",
		"        intake = new Intake(new IntakeIO() {});\n        climber = new Climber(new ClimberIO() {});\n        break;",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing case lines:\n%s", want)
		}
	}
	if count := strings.Count(s, "break;"); count != 3 {
		t.Errorf("break count = %d, want 3", count)
	}
}

func TestInject_RejectsDuplicate(t *testing.T) {
	inj := loadFixtures(t)
	err := inj.Inject(injector.SubsystemInjection{Name: "Shooter", NameLower: "shooter", Package: "frc.robot"})
	if err == nil {
		t.Fatal("expected error injecting a mechanism the container already has")
	}
	if strings.Count(read(t, inj.RobotContainerPath), "import frc.robot.subsystems.shooter.Shooter;") != 1 {
		t.Error("failed Inject modified RobotContainer.java")
	}
}

func TestInject_NoSwitchBlock_ReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "RobotContainer.java")
	os.WriteFile(path, []byte("import a.B;\npublic class RobotContainer {\n  public final Drive drive;\n}\n"), 0644)
	err := injector.New(path, "").Inject(climber)
	if err == nil {
		t.Error("expected error for missing switch block")
	}
}

func TestEject_RoundTrip(t *testing.T) {
	inj := loadFixtures(t)
	original := read(t, inj.RobotContainerPath)
	if err := inj.Inject(climber); err != nil {
		t.Fatalf("Inject() error: %v", err)
	}
	if err := inj.Eject(climber); err != nil {
		t.Fatalf("Eject() error: %v", err)
	}
	if read(t, inj.RobotContainerPath) != original {
		t.Error("RobotContainer.java after Inject+Eject does not match original")
	}
}

func TestEject_NothingInjected_ReturnsError(t *testing.T) {
	inj := loadFixtures(t)
	if err := inj.Eject(climber); err == nil {
		t.Error("expected error when nothing to eject")
	}
}

func TestInjectPeriodic_AfterLastPeriodicCall(t *testing.T) {
	inj := loadFixtures(t)
	if err := inj.InjectPeriodic("climber"); err != nil {
		t.Fatalf("InjectPeriodic() error: %v", err)
	}
	want := "    container.superstructure.periodic();\n    container.climber.periodic();\n"
	if !strings.Contains(read(t, inj.RobotPath), want) {
		t.Error("climber.periodic() not placed after the last periodic() call")
	}
}

func TestInjectPeriodic_NoExistingCalls_GoesBeforeScheduler(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Robot.java")
	os.WriteFile(path, []byte(`public class Robot {
  @Override
  public void robotPeriodic() {
    Scheduler.getDefault().run();
  }
}
`), 0644)
	inj := injector.New("", path)
	if err := inj.InjectPeriodic("climber"); err != nil {
		t.Fatalf("InjectPeriodic() error: %v", err)
	}
	if !strings.Contains(read(t, path), "    container.climber.periodic();\n    Scheduler.getDefault().run();") {
		t.Errorf("periodic() not placed before scheduler:\n%s", read(t, path))
	}
}

func TestInjectPeriodic_RejectsDuplicate(t *testing.T) {
	inj := loadFixtures(t)
	if err := inj.InjectPeriodic("shooter"); err == nil {
		t.Error("expected error for a mechanism already ticked in robotPeriodic()")
	}
}

func TestEjectPeriodic_RoundTrip(t *testing.T) {
	inj := loadFixtures(t)
	original := read(t, inj.RobotPath)
	inj.InjectPeriodic("climber")
	if err := inj.EjectPeriodic("climber"); err != nil {
		t.Fatalf("EjectPeriodic() error: %v", err)
	}
	if read(t, inj.RobotPath) != original {
		t.Error("Robot.java after InjectPeriodic+EjectPeriodic does not match original")
	}
}
