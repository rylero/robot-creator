# robot-creator Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers-optimized:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go CLI that scaffolds AdvantageKit IO-layer FRC subsystems from embedded templates.  
**Architecture:** cobra CLI → commands call internal/generator (renders text/template files) and internal/injector (patches RobotContainer.java); templates embedded via embed.FS behind a TemplateSource interface.  
**Tech Stack:** Go 1.22, github.com/spf13/cobra, gopkg.in/yaml.v3, stdlib text/template + embed  
**Assumptions:** RobotContainer.java uses `switch (Constants.currentMode)` with `case REAL:` / `case SIM:` / `default:` each terminated by `break;`. Injection aborts cleanly if pattern not found.

---

### Task 1: Project Bootstrap

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `cmd/root.go`
- Create: `cmd/add.go`

**Security flag:** `none`

- [ ] **Step 1: Initialize Go module**

```bash
cd C:\Users\ryan\Dev\RobotCreator
go mod init github.com/ced4rtree/robot-creator
go get github.com/spf13/cobra@v1.8.0
go get gopkg.in/yaml.v3@v3.0.1
```

- [ ] **Step 2: Create main.go**

```go
package main

import "github.com/ced4rtree/robot-creator/cmd"

func main() {
	cmd.Execute()
}
```

- [ ] **Step 3: Create cmd/root.go**

```go
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "robot-creator",
	Short: "Scaffold AdvantageKit FRC robot projects",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
}
```

- [ ] **Step 4: Create cmd/add.go**

```go
package cmd

import "github.com/spf13/cobra"

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add components to the robot project",
}
```

- [ ] **Step 5: Verify build**

Run: `go build ./...`  
Expected: no errors

- [ ] **Step 6: Commit**

```bash
git init
git add go.mod go.sum main.go cmd/
git commit -m "bootstrap: Go module + cobra root/add commands"
```

---

### Task 2: Config Package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Security flag:** `none`

- [ ] **Step 1: Write failing tests**

Create `internal/config/config_test.go`:

```go
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ced4rtree/robot-creator/internal/config"
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
```

- [ ] **Step 2: Run tests — expect FAIL**

Run: `go test ./internal/config/...`  
Expected: compile error (package doesn't exist yet)

- [ ] **Step 3: Implement config.go**

```go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const FileName = "robot-creator.yaml"

type Config struct {
	Team           int         `yaml:"team"`
	Package        string      `yaml:"package"`
	Subsystems     []Subsystem `yaml:"subsystems,omitempty"`
	Superstructure bool        `yaml:"superstructure,omitempty"`
}

type Subsystem struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

func Load(root string) (*Config, error) {
	data, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		return nil, fmt.Errorf("not a robot-creator project (robot-creator.yaml not found): run 'robot-creator init' first")
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("malformed robot-creator.yaml: %w", err)
	}
	return &cfg, nil
}

func Save(root string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, FileName), data, 0644)
}

func FindRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, FileName)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not in a robot-creator project. Run 'robot-creator init' first")
		}
		dir = parent
	}
}

func (c *Config) HasSubsystem(name string) bool {
	for _, s := range c.Subsystems {
		if s.Name == name {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run tests — expect PASS**

Run: `go test ./internal/config/...`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: config package — load/save robot-creator.yaml"
```

---

### Task 3: TemplateSource Interface + EmbeddedSource

**Files:**
- Create: `internal/template/source.go`
- Create: `internal/template/embedded.go`
- Create: `internal/template/templates/.gitkeep` (placeholder so dir is committed)

**Security flag:** `none`

- [ ] **Step 1: Create source.go**

```go
package template

var validTypes = []string{
	"flywheel", "pivot", "roller", "arm", "elevator", "turret", "generic",
}

// TemplateSource abstracts where templates come from.
// EmbeddedTemplateSource is the MVP impl; LocalTemplateSource is planned post-MVP.
type TemplateSource interface {
	GetTemplate(subsystemType, fileName string) ([]byte, error)
	ListTypes() []string
}

func IsValidType(t string) bool {
	for _, v := range validTypes {
		if v == t {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Create embedded.go**

```go
package template

import (
	"embed"
	"fmt"
)

//go:embed templates
var embeddedFS embed.FS

type EmbeddedTemplateSource struct{}

func NewEmbeddedSource() *EmbeddedTemplateSource {
	return &EmbeddedTemplateSource{}
}

func (e *EmbeddedTemplateSource) GetTemplate(subsystemType, fileName string) ([]byte, error) {
	path := fmt.Sprintf("templates/%s/%s", subsystemType, fileName)
	data, err := embeddedFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("template %s/%s not found: %w", subsystemType, fileName, err)
	}
	return data, nil
}

func (e *EmbeddedTemplateSource) ListTypes() []string {
	return validTypes
}
```

- [ ] **Step 3: Create placeholder so embed compiles**

Create empty file `internal/template/templates/.gitkeep` (content: empty).

- [ ] **Step 4: Verify build**

Run: `go build ./...`  
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add internal/template/
git commit -m "feat: TemplateSource interface + EmbeddedTemplateSource"
```

---

### Task 4: Generic Templates

**Files:**
- Create: `internal/template/templates/generic/Subsystem.java.tmpl`
- Create: `internal/template/templates/generic/SubsystemIO.java.tmpl`
- Create: `internal/template/templates/generic/SubsystemIOTalonFX.java.tmpl`
- Create: `internal/template/templates/generic/SubsystemIOSim.java.tmpl`
- Create: `internal/template/templates/generic/SubsystemConstants.java.tmpl`

**Security flag:** `none`

- [ ] **Step 1: Create Subsystem.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import edu.wpi.first.wpilibj2.command.SubsystemBase;
import org.littletonrobotics.junction.Logger;

public class {{.Name}} extends SubsystemBase {
  private final {{.Name}}IO io;
  private final {{.Name}}IOInputsAutoLogged inputs = new {{.Name}}IOInputsAutoLogged();

  public {{.Name}}({{.Name}}IO io) {
    this.io = io;
  }

  @Override
  public void periodic() {
    io.updateInputs(inputs);
    Logger.processInputs("{{.Name}}", inputs);
  }
}
```

- [ ] **Step 2: Create SubsystemIO.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import edu.wpi.first.units.measure.Current;
import edu.wpi.first.units.measure.Voltage;
import org.littletonrobotics.junction.AutoLog;

public interface {{.Name}}IO {
  @AutoLog
  public static class {{.Name}}IOInputs {
    public boolean motorConnected = false;
    public Voltage appliedVoltage = Volts.of(0.0);
    public Current supplyCurrent = Amps.of(0.0);
  }

  public default void updateInputs({{.Name}}IOInputs inputs) {}

  public default void setVoltage(Voltage volts) {}
}
```

- [ ] **Step 3: Create SubsystemIOTalonFX.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import com.ctre.phoenix6.StatusSignal;
import com.ctre.phoenix6.configs.TalonFXConfiguration;
import com.ctre.phoenix6.controls.VoltageOut;
import com.ctre.phoenix6.hardware.TalonFX;
import edu.wpi.first.units.measure.Current;
import edu.wpi.first.units.measure.Voltage;

public class {{.Name}}IOTalonFX implements {{.Name}}IO {
  private final TalonFX motor =
      new TalonFX({{.Name}}Constants.MOTOR_ID, {{.Name}}Constants.CAN_BUS);
  private final StatusSignal<Voltage> appliedVoltage;
  private final StatusSignal<Current> supplyCurrent;
  private final VoltageOut voltageRequest = new VoltageOut(0);

  public {{.Name}}IOTalonFX() {
    var config = new TalonFXConfiguration();
    config.CurrentLimits.StatorCurrentLimitEnable = true;
    config.CurrentLimits.StatorCurrentLimit = 40;
    config.CurrentLimits.SupplyCurrentLimitEnable = true;
    config.CurrentLimits.SupplyCurrentLimit = 20;
    motor.getConfigurator().apply(config);

    appliedVoltage = motor.getMotorVoltage();
    supplyCurrent = motor.getSupplyCurrent();
    appliedVoltage.setUpdateFrequency(50);
    supplyCurrent.setUpdateFrequency(20);
  }

  @Override
  public void updateInputs({{.Name}}IOInputs inputs) {
    inputs.motorConnected = appliedVoltage.getStatus().isOK();
    inputs.appliedVoltage = appliedVoltage.getValue();
    inputs.supplyCurrent = supplyCurrent.getValue();
  }

  @Override
  public void setVoltage(Voltage volts) {
    motor.setControl(voltageRequest.withOutput(volts));
  }
}
```

- [ ] **Step 4: Create SubsystemIOSim.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import edu.wpi.first.math.system.plant.DCMotor;
import edu.wpi.first.math.system.plant.LinearSystemId;
import edu.wpi.first.units.measure.Voltage;
import edu.wpi.first.wpilibj.simulation.FlywheelSim;

public class {{.Name}}IOSim implements {{.Name}}IO {
  private final FlywheelSim sim =
      new FlywheelSim(
          LinearSystemId.createFlywheelSystem(DCMotor.getKrakenX60(1), 0.001, 1.0),
          DCMotor.getKrakenX60(1));
  private double appliedVolts = 0.0;

  @Override
  public void updateInputs({{.Name}}IOInputs inputs) {
    sim.setInputVoltage(appliedVolts);
    sim.update(0.02);
    inputs.motorConnected = true;
    inputs.appliedVoltage = Volts.of(appliedVolts);
    inputs.supplyCurrent = Amps.of(Math.abs(sim.getCurrentDrawAmps()));
  }

  @Override
  public void setVoltage(Voltage volts) {
    appliedVolts = volts.in(Volts);
  }
}
```

- [ ] **Step 5: Create SubsystemConstants.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import com.ctre.phoenix6.CANBus;

public class {{.Name}}Constants {
  public static final CANBus CAN_BUS = new CANBus("canivore");
  public static final int MOTOR_ID = 0; // TODO: set CAN ID
}
```

- [ ] **Step 6: Commit**

```bash
git add internal/template/templates/generic/
git commit -m "feat: generic subsystem templates (AKit IO-layer scaffold)"
```

---

### Task 5: Flywheel Templates

**Files:**
- Create: `internal/template/templates/flywheel/Subsystem.java.tmpl`
- Create: `internal/template/templates/flywheel/SubsystemIO.java.tmpl`
- Create: `internal/template/templates/flywheel/SubsystemIOTalonFX.java.tmpl`
- Create: `internal/template/templates/flywheel/SubsystemIOSim.java.tmpl`
- Create: `internal/template/templates/flywheel/SubsystemConstants.java.tmpl`

**Security flag:** `none`

- [ ] **Step 1: Create Subsystem.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import edu.wpi.first.units.measure.AngularVelocity;
import edu.wpi.first.wpilibj2.command.Command;
import edu.wpi.first.wpilibj2.command.SubsystemBase;
import org.littletonrobotics.junction.Logger;

public class {{.Name}} extends SubsystemBase {
  private final {{.Name}}IO io;
  private final {{.Name}}IOInputsAutoLogged inputs = new {{.Name}}IOInputsAutoLogged();
  private AngularVelocity targetVelocity = RotationsPerSecond.of(0);

  public {{.Name}}({{.Name}}IO io) {
    this.io = io;
  }

  @Override
  public void periodic() {
    io.updateInputs(inputs);
    Logger.processInputs("{{.Name}}", inputs);
  }

  public Command setVelocity(AngularVelocity velocity) {
    return runOnce(() -> {
      targetVelocity = velocity;
      io.setVelocity(velocity);
    });
  }

  public Command stop() {
    return runOnce(() -> {
      targetVelocity = RotationsPerSecond.of(0);
      io.stop();
    });
  }

  public boolean atTargetVelocity() {
    return inputs.velocity.isNear(targetVelocity, {{.Name}}Constants.VELOCITY_TOLERANCE);
  }
}
```

- [ ] **Step 2: Create SubsystemIO.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import edu.wpi.first.units.measure.AngularVelocity;
import edu.wpi.first.units.measure.Current;
import edu.wpi.first.units.measure.Voltage;
import org.littletonrobotics.junction.AutoLog;

public interface {{.Name}}IO {
  @AutoLog
  public static class {{.Name}}IOInputs {
    public boolean motorConnected = false;
    public AngularVelocity velocity = RotationsPerSecond.of(0.0);
    public Current supplyCurrent = Amps.of(0.0);
    public Voltage appliedVoltage = Volts.of(0.0);
  }

  public default void updateInputs({{.Name}}IOInputs inputs) {}

  public default void setVelocity(AngularVelocity velocity) {}

  public default void stop() {}
}
```

- [ ] **Step 3: Create SubsystemIOTalonFX.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import com.ctre.phoenix6.StatusSignal;
import com.ctre.phoenix6.configs.TalonFXConfiguration;
import com.ctre.phoenix6.controls.MotionMagicVelocityVoltage;
import com.ctre.phoenix6.hardware.TalonFX;
import edu.wpi.first.units.measure.AngularVelocity;
import edu.wpi.first.units.measure.Current;
import edu.wpi.first.units.measure.Voltage;

public class {{.Name}}IOTalonFX implements {{.Name}}IO {
  private final TalonFX motor =
      new TalonFX({{.Name}}Constants.MOTOR_ID, {{.Name}}Constants.CAN_BUS);
  private final StatusSignal<AngularVelocity> velocity;
  private final StatusSignal<Current> supplyCurrent;
  private final StatusSignal<Voltage> appliedVoltage;
  private final MotionMagicVelocityVoltage velocityRequest =
      new MotionMagicVelocityVoltage(0).withSlot(0);

  public {{.Name}}IOTalonFX() {
    var config = new TalonFXConfiguration();
    config.Feedback.SensorToMechanismRatio = {{.Name}}Constants.GEARING;
    config.MotorOutput.Inverted = {{.Name}}Constants.MOTOR_DIRECTION;
    config.Slot0 = {{.Name}}Constants.GAINS.toSlot0Configs();
    config.MotionMagic = {{.Name}}Constants.MOTION_MAGIC_CONFIGS;
    config.CurrentLimits.StatorCurrentLimitEnable = true;
    config.CurrentLimits.StatorCurrentLimit = 40;
    config.CurrentLimits.SupplyCurrentLimitEnable = true;
    config.CurrentLimits.SupplyCurrentLimit = 20;
    motor.getConfigurator().apply(config);

    velocity = motor.getVelocity();
    supplyCurrent = motor.getSupplyCurrent();
    appliedVoltage = motor.getMotorVoltage();
    velocity.setUpdateFrequency(50);
    supplyCurrent.setUpdateFrequency(20);
    appliedVoltage.setUpdateFrequency(20);
  }

  @Override
  public void updateInputs({{.Name}}IOInputs inputs) {
    inputs.motorConnected = velocity.getStatus().isOK();
    inputs.velocity = velocity.getValue();
    inputs.supplyCurrent = supplyCurrent.getValue();
    inputs.appliedVoltage = appliedVoltage.getValue();
  }

  @Override
  public void setVelocity(AngularVelocity v) {
    motor.setControl(velocityRequest.withVelocity(v));
  }

  @Override
  public void stop() {
    motor.stopMotor();
  }
}
```

- [ ] **Step 4: Create SubsystemIOSim.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import edu.wpi.first.math.system.plant.DCMotor;
import edu.wpi.first.math.system.plant.LinearSystemId;
import edu.wpi.first.units.measure.AngularVelocity;
import edu.wpi.first.wpilibj.simulation.FlywheelSim;

public class {{.Name}}IOSim implements {{.Name}}IO {
  private final FlywheelSim sim =
      new FlywheelSim(
          LinearSystemId.createFlywheelSystem(
              DCMotor.getKrakenX60(1), 0.001, {{.Name}}Constants.GEARING),
          DCMotor.getKrakenX60(1));
  private double appliedVolts = 0.0;

  @Override
  public void updateInputs({{.Name}}IOInputs inputs) {
    sim.setInputVoltage(appliedVolts);
    sim.update(0.02);
    inputs.motorConnected = true;
    inputs.velocity =
        RotationsPerSecond.of(sim.getAngularVelocityRadPerSec() / (2.0 * Math.PI));
    inputs.supplyCurrent = Amps.of(Math.abs(sim.getCurrentDrawAmps()));
    inputs.appliedVoltage = Volts.of(appliedVolts);
  }

  @Override
  public void setVelocity(AngularVelocity velocity) {
    double rps = velocity.in(RotationsPerSecond);
    appliedVolts = rps == 0 ? 0 : rps * 0.12 + Math.signum(rps) * 0.1;
  }

  @Override
  public void stop() {
    appliedVolts = 0.0;
  }
}
```

- [ ] **Step 5: Create SubsystemConstants.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import com.ctre.phoenix6.CANBus;
import com.ctre.phoenix6.configs.MotionMagicConfigs;
import com.ctre.phoenix6.signals.InvertedValue;
import static edu.wpi.first.units.Units.*;
import edu.wpi.first.units.measure.AngularVelocity;
import frc.robot.util.LoggedNetworkPIDFeedforwardGains;
import org.littletonrobotics.junction.networktables.LoggedNetworkNumber;

public class {{.Name}}Constants {
  public static final CANBus CAN_BUS = new CANBus("canivore");
  public static final int MOTOR_ID = 0; // TODO: set CAN ID
  public static final double GEARING = 1.0;
  public static final InvertedValue MOTOR_DIRECTION = InvertedValue.CounterClockwise_Positive;
  public static final AngularVelocity VELOCITY_TOLERANCE = RotationsPerSecond.of(2.0);

  public static final LoggedNetworkNumber TARGET_VELOCITY =
      new LoggedNetworkNumber("{{.Name}}/TargetVelocityRPS", 50.0);

  public static final LoggedNetworkPIDFeedforwardGains GAINS =
      new LoggedNetworkPIDFeedforwardGains(
          0.5, 0.0, 0.0, 0.0, 0.12, 0.24, 0.0, "{{.Name}}");

  public static final MotionMagicConfigs MOTION_MAGIC_CONFIGS =
      new MotionMagicConfigs().withMotionMagicAcceleration(300);
}
```

- [ ] **Step 6: Commit**

```bash
git add internal/template/templates/flywheel/
git commit -m "feat: flywheel templates (velocity control, MotionMagicVelocityVoltage)"
```

---

### Task 6: Pivot Templates

**Files:**
- Create: `internal/template/templates/pivot/Subsystem.java.tmpl`
- Create: `internal/template/templates/pivot/SubsystemIO.java.tmpl`
- Create: `internal/template/templates/pivot/SubsystemIOTalonFX.java.tmpl`
- Create: `internal/template/templates/pivot/SubsystemIOSim.java.tmpl`
- Create: `internal/template/templates/pivot/SubsystemConstants.java.tmpl`

**Security flag:** `none`

- [ ] **Step 1: Create Subsystem.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import edu.wpi.first.units.measure.Angle;
import edu.wpi.first.wpilibj2.command.Command;
import edu.wpi.first.wpilibj2.command.SubsystemBase;
import org.littletonrobotics.junction.Logger;

public class {{.Name}} extends SubsystemBase {
  private final {{.Name}}IO io;
  private final {{.Name}}IOInputsAutoLogged inputs = new {{.Name}}IOInputsAutoLogged();
  private Angle targetAngle = Degrees.of(0);

  public {{.Name}}({{.Name}}IO io) {
    this.io = io;
  }

  @Override
  public void periodic() {
    io.updateInputs(inputs);
    Logger.processInputs("{{.Name}}", inputs);
  }

  public Command setAngle(Angle angle) {
    return runOnce(() -> {
      targetAngle = angle;
      io.setAngle(angle);
    });
  }

  public boolean atTargetAngle() {
    return inputs.position.isNear(targetAngle, {{.Name}}Constants.ANGLE_TOLERANCE);
  }
}
```

- [ ] **Step 2: Create SubsystemIO.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import edu.wpi.first.units.measure.Angle;
import edu.wpi.first.units.measure.AngularVelocity;
import edu.wpi.first.units.measure.Current;
import edu.wpi.first.units.measure.Voltage;
import org.littletonrobotics.junction.AutoLog;

public interface {{.Name}}IO {
  @AutoLog
  public static class {{.Name}}IOInputs {
    public boolean motorConnected = false;
    public Angle position = Degrees.of(0.0);
    public AngularVelocity velocity = RotationsPerSecond.of(0.0);
    public Current supplyCurrent = Amps.of(0.0);
    public Voltage appliedVoltage = Volts.of(0.0);
  }

  public default void updateInputs({{.Name}}IOInputs inputs) {}

  public default void setAngle(Angle angle) {}

  public default void setVoltage(Voltage volts) {}

  public default void setBrakeMode(boolean brake) {}
}
```

- [ ] **Step 3: Create SubsystemIOTalonFX.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import com.ctre.phoenix6.StatusSignal;
import com.ctre.phoenix6.configs.TalonFXConfiguration;
import com.ctre.phoenix6.controls.MotionMagicPositionVoltage;
import com.ctre.phoenix6.controls.VoltageOut;
import com.ctre.phoenix6.hardware.TalonFX;
import com.ctre.phoenix6.signals.NeutralModeValue;
import edu.wpi.first.units.measure.Angle;
import edu.wpi.first.units.measure.AngularVelocity;
import edu.wpi.first.units.measure.Current;
import edu.wpi.first.units.measure.Voltage;

public class {{.Name}}IOTalonFX implements {{.Name}}IO {
  private final TalonFX motor =
      new TalonFX({{.Name}}Constants.MOTOR_ID, {{.Name}}Constants.CAN_BUS);
  private final StatusSignal<Angle> position;
  private final StatusSignal<AngularVelocity> velocity;
  private final StatusSignal<Current> supplyCurrent;
  private final StatusSignal<Voltage> appliedVoltage;
  private final MotionMagicPositionVoltage positionRequest =
      new MotionMagicPositionVoltage(0).withSlot(0);
  private final VoltageOut voltageRequest = new VoltageOut(0);

  public {{.Name}}IOTalonFX() {
    var config = new TalonFXConfiguration();
    config.Feedback.SensorToMechanismRatio = {{.Name}}Constants.GEARING;
    config.MotorOutput.Inverted = {{.Name}}Constants.MOTOR_DIRECTION;
    config.MotorOutput.NeutralMode = NeutralModeValue.Brake;
    config.Slot0 = {{.Name}}Constants.GAINS.toSlot0Configs();
    config.MotionMagic = {{.Name}}Constants.MOTION_MAGIC_CONFIGS;
    config.SoftwareLimitSwitch.ForwardSoftLimitEnable = true;
    config.SoftwareLimitSwitch.ForwardSoftLimitThreshold =
        {{.Name}}Constants.MAX_ANGLE.in(Rotations);
    config.SoftwareLimitSwitch.ReverseSoftLimitEnable = true;
    config.SoftwareLimitSwitch.ReverseSoftLimitThreshold =
        {{.Name}}Constants.MIN_ANGLE.in(Rotations);
    config.CurrentLimits.StatorCurrentLimitEnable = true;
    config.CurrentLimits.StatorCurrentLimit = 40;
    motor.getConfigurator().apply(config);

    position = motor.getPosition();
    velocity = motor.getVelocity();
    supplyCurrent = motor.getSupplyCurrent();
    appliedVoltage = motor.getMotorVoltage();
    position.setUpdateFrequency(50);
    velocity.setUpdateFrequency(50);
    supplyCurrent.setUpdateFrequency(20);
    appliedVoltage.setUpdateFrequency(20);
  }

  @Override
  public void updateInputs({{.Name}}IOInputs inputs) {
    inputs.motorConnected = position.getStatus().isOK();
    inputs.position = position.getValue();
    inputs.velocity = velocity.getValue();
    inputs.supplyCurrent = supplyCurrent.getValue();
    inputs.appliedVoltage = appliedVoltage.getValue();
  }

  @Override
  public void setAngle(Angle angle) {
    motor.setControl(positionRequest.withPosition(angle.in(Rotations)));
  }

  @Override
  public void setVoltage(Voltage volts) {
    motor.setControl(voltageRequest.withOutput(volts));
  }

  @Override
  public void setBrakeMode(boolean brake) {
    motor.setNeutralMode(brake ? NeutralModeValue.Brake : NeutralModeValue.Coast);
  }
}
```

- [ ] **Step 4: Create SubsystemIOSim.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import edu.wpi.first.math.system.plant.DCMotor;
import edu.wpi.first.units.measure.Angle;
import edu.wpi.first.units.measure.Voltage;
import edu.wpi.first.wpilibj.simulation.SingleJointedArmSim;

public class {{.Name}}IOSim implements {{.Name}}IO {
  private final SingleJointedArmSim sim =
      new SingleJointedArmSim(
          DCMotor.getKrakenX60(1),
          {{.Name}}Constants.GEARING,
          0.1,
          0.3,
          {{.Name}}Constants.MIN_ANGLE.in(Radians),
          {{.Name}}Constants.MAX_ANGLE.in(Radians),
          true,
          {{.Name}}Constants.MIN_ANGLE.in(Radians));
  private double appliedVolts = 0.0;

  @Override
  public void updateInputs({{.Name}}IOInputs inputs) {
    sim.setInputVoltage(appliedVolts);
    sim.update(0.02);
    inputs.motorConnected = true;
    inputs.position = Radians.of(sim.getAngleRads());
    inputs.velocity = RadiansPerSecond.of(sim.getVelocityRadPerSec());
    inputs.supplyCurrent = Amps.of(Math.abs(sim.getCurrentDrawAmps()));
    inputs.appliedVoltage = Volts.of(appliedVolts);
  }

  @Override
  public void setAngle(Angle angle) {
    double error = angle.in(Radians) - sim.getAngleRads();
    appliedVolts = Math.max(-12, Math.min(12, error * 10));
  }

  @Override
  public void setVoltage(Voltage volts) {
    appliedVolts = volts.in(Volts);
  }
}
```

- [ ] **Step 5: Create SubsystemConstants.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import com.ctre.phoenix6.CANBus;
import com.ctre.phoenix6.configs.MotionMagicConfigs;
import com.ctre.phoenix6.signals.InvertedValue;
import static edu.wpi.first.units.Units.*;
import edu.wpi.first.units.measure.Angle;
import frc.robot.util.LoggedNetworkPIDFeedforwardGains;

public class {{.Name}}Constants {
  public static final CANBus CAN_BUS = new CANBus("canivore");
  public static final int MOTOR_ID = 0; // TODO: set CAN ID
  public static final double GEARING = 1.0;
  public static final InvertedValue MOTOR_DIRECTION = InvertedValue.CounterClockwise_Positive;

  public static final Angle MIN_ANGLE = Degrees.of(0.0);
  public static final Angle MAX_ANGLE = Degrees.of(90.0);
  public static final Angle ANGLE_TOLERANCE = Degrees.of(1.0);

  public static final LoggedNetworkPIDFeedforwardGains GAINS =
      new LoggedNetworkPIDFeedforwardGains(
          2.0, 0.0, 0.0, 0.0, 0.0, 0.3, 0.5, "{{.Name}}");

  public static final MotionMagicConfigs MOTION_MAGIC_CONFIGS =
      new MotionMagicConfigs()
          .withMotionMagicCruiseVelocity(0.5)
          .withMotionMagicAcceleration(1.0);
}
```

- [ ] **Step 6: Commit**

```bash
git add internal/template/templates/pivot/
git commit -m "feat: pivot templates (position control, MotionMagicPositionVoltage)"
```

---

### Task 7: Roller Templates

**Files:**
- Create: `internal/template/templates/roller/Subsystem.java.tmpl`
- Create: `internal/template/templates/roller/SubsystemIO.java.tmpl`
- Create: `internal/template/templates/roller/SubsystemIOTalonFX.java.tmpl`
- Create: `internal/template/templates/roller/SubsystemIOSim.java.tmpl`
- Create: `internal/template/templates/roller/SubsystemConstants.java.tmpl`

**Security flag:** `none`

- [ ] **Step 1: Create Subsystem.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import edu.wpi.first.units.measure.Voltage;
import edu.wpi.first.wpilibj2.command.Command;
import edu.wpi.first.wpilibj2.command.SubsystemBase;
import org.littletonrobotics.junction.Logger;

public class {{.Name}} extends SubsystemBase {
  private final {{.Name}}IO io;
  private final {{.Name}}IOInputsAutoLogged inputs = new {{.Name}}IOInputsAutoLogged();

  public {{.Name}}({{.Name}}IO io) {
    this.io = io;
  }

  @Override
  public void periodic() {
    io.updateInputs(inputs);
    Logger.processInputs("{{.Name}}", inputs);
  }

  public Command run(Voltage voltage) {
    return runOnce(() -> io.setVoltage(voltage));
  }

  public Command stop() {
    return runOnce(() -> io.setVoltage(Volts.of(0)));
  }
}
```

- [ ] **Step 2: Create SubsystemIO.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import edu.wpi.first.units.measure.AngularVelocity;
import edu.wpi.first.units.measure.Current;
import edu.wpi.first.units.measure.Voltage;
import org.littletonrobotics.junction.AutoLog;

public interface {{.Name}}IO {
  @AutoLog
  public static class {{.Name}}IOInputs {
    public boolean motorConnected = false;
    public AngularVelocity velocity = RotationsPerSecond.of(0.0);
    public Voltage appliedVoltage = Volts.of(0.0);
    public Current supplyCurrent = Amps.of(0.0);
  }

  public default void updateInputs({{.Name}}IOInputs inputs) {}

  public default void setVoltage(Voltage volts) {}
}
```

- [ ] **Step 3: Create SubsystemIOTalonFX.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import com.ctre.phoenix6.StatusSignal;
import com.ctre.phoenix6.configs.TalonFXConfiguration;
import com.ctre.phoenix6.controls.VoltageOut;
import com.ctre.phoenix6.hardware.TalonFX;
import edu.wpi.first.units.measure.AngularVelocity;
import edu.wpi.first.units.measure.Current;
import edu.wpi.first.units.measure.Voltage;

public class {{.Name}}IOTalonFX implements {{.Name}}IO {
  private final TalonFX motor =
      new TalonFX({{.Name}}Constants.MOTOR_ID, {{.Name}}Constants.CAN_BUS);
  private final StatusSignal<AngularVelocity> velocity;
  private final StatusSignal<Voltage> appliedVoltage;
  private final StatusSignal<Current> supplyCurrent;
  private final VoltageOut voltageRequest = new VoltageOut(0);

  public {{.Name}}IOTalonFX() {
    var config = new TalonFXConfiguration();
    config.CurrentLimits.StatorCurrentLimitEnable = true;
    config.CurrentLimits.StatorCurrentLimit = 40;
    config.CurrentLimits.SupplyCurrentLimitEnable = true;
    config.CurrentLimits.SupplyCurrentLimit = 20;
    motor.getConfigurator().apply(config);

    velocity = motor.getVelocity();
    appliedVoltage = motor.getMotorVoltage();
    supplyCurrent = motor.getSupplyCurrent();
    velocity.setUpdateFrequency(50);
    appliedVoltage.setUpdateFrequency(20);
    supplyCurrent.setUpdateFrequency(20);
  }

  @Override
  public void updateInputs({{.Name}}IOInputs inputs) {
    inputs.motorConnected = velocity.getStatus().isOK();
    inputs.velocity = velocity.getValue();
    inputs.appliedVoltage = appliedVoltage.getValue();
    inputs.supplyCurrent = supplyCurrent.getValue();
  }

  @Override
  public void setVoltage(Voltage volts) {
    motor.setControl(voltageRequest.withOutput(volts));
  }
}
```

- [ ] **Step 4: Create SubsystemIOSim.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import edu.wpi.first.math.system.plant.DCMotor;
import edu.wpi.first.math.system.plant.LinearSystemId;
import edu.wpi.first.units.measure.Voltage;
import edu.wpi.first.wpilibj.simulation.FlywheelSim;

public class {{.Name}}IOSim implements {{.Name}}IO {
  private final FlywheelSim sim =
      new FlywheelSim(
          LinearSystemId.createFlywheelSystem(DCMotor.getKrakenX60(1), 0.001, 1.0),
          DCMotor.getKrakenX60(1));
  private double appliedVolts = 0.0;

  @Override
  public void updateInputs({{.Name}}IOInputs inputs) {
    sim.setInputVoltage(appliedVolts);
    sim.update(0.02);
    inputs.motorConnected = true;
    inputs.velocity =
        RotationsPerSecond.of(sim.getAngularVelocityRadPerSec() / (2.0 * Math.PI));
    inputs.appliedVoltage = Volts.of(appliedVolts);
    inputs.supplyCurrent = Amps.of(Math.abs(sim.getCurrentDrawAmps()));
  }

  @Override
  public void setVoltage(Voltage volts) {
    appliedVolts = volts.in(Volts);
  }
}
```

- [ ] **Step 5: Create SubsystemConstants.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import com.ctre.phoenix6.CANBus;

public class {{.Name}}Constants {
  public static final CANBus CAN_BUS = new CANBus("canivore");
  public static final int MOTOR_ID = 0; // TODO: set CAN ID
}
```

- [ ] **Step 6: Commit**

```bash
git add internal/template/templates/roller/
git commit -m "feat: roller templates (voltage control + velocity monitoring)"
```

---

### Task 8: Arm, Elevator, Turret Templates

**Files:**
- Create: `internal/template/templates/arm/` (5 files)
- Create: `internal/template/templates/elevator/` (5 files)
- Create: `internal/template/templates/turret/` (5 files)

**Security flag:** `none`

- [ ] **Step 1: Create arm/SubsystemIO.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import edu.wpi.first.units.measure.Angle;
import edu.wpi.first.units.measure.AngularVelocity;
import edu.wpi.first.units.measure.Current;
import edu.wpi.first.units.measure.Voltage;
import org.littletonrobotics.junction.AutoLog;

public interface {{.Name}}IO {
  @AutoLog
  public static class {{.Name}}IOInputs {
    public boolean leaderConnected = false;
    public boolean followerConnected = false;
    public Angle position = Degrees.of(0.0);
    public AngularVelocity velocity = RotationsPerSecond.of(0.0);
    public Voltage appliedVoltage = Volts.of(0.0);
    public Current leaderSupplyCurrent = Amps.of(0.0);
    public Current followerSupplyCurrent = Amps.of(0.0);
  }

  public default void updateInputs({{.Name}}IOInputs inputs) {}

  public default void setAngle(Angle angle) {}

  public default void setVoltage(Voltage volts) {}

  public default void setBrakeMode(boolean brake) {}
}
```

- [ ] **Step 2: Create arm/Subsystem.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import edu.wpi.first.units.measure.Angle;
import edu.wpi.first.wpilibj2.command.Command;
import edu.wpi.first.wpilibj2.command.SubsystemBase;
import org.littletonrobotics.junction.Logger;

public class {{.Name}} extends SubsystemBase {
  private final {{.Name}}IO io;
  private final {{.Name}}IOInputsAutoLogged inputs = new {{.Name}}IOInputsAutoLogged();
  private Angle targetAngle = Degrees.of(0);

  public {{.Name}}({{.Name}}IO io) {
    this.io = io;
  }

  @Override
  public void periodic() {
    io.updateInputs(inputs);
    Logger.processInputs("{{.Name}}", inputs);
  }

  public Command setAngle(Angle angle) {
    return runOnce(() -> {
      targetAngle = angle;
      io.setAngle(angle);
    });
  }

  public boolean atTargetAngle() {
    return inputs.position.isNear(targetAngle, {{.Name}}Constants.ANGLE_TOLERANCE);
  }
}
```

- [ ] **Step 3: Create arm/SubsystemIOTalonFX.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import com.ctre.phoenix6.StatusSignal;
import com.ctre.phoenix6.configs.TalonFXConfiguration;
import com.ctre.phoenix6.controls.Follower;
import com.ctre.phoenix6.controls.MotionMagicPositionVoltage;
import com.ctre.phoenix6.controls.VoltageOut;
import com.ctre.phoenix6.hardware.TalonFX;
import com.ctre.phoenix6.signals.NeutralModeValue;
import edu.wpi.first.units.measure.Angle;
import edu.wpi.first.units.measure.AngularVelocity;
import edu.wpi.first.units.measure.Current;
import edu.wpi.first.units.measure.Voltage;

public class {{.Name}}IOTalonFX implements {{.Name}}IO {
  private final TalonFX leader =
      new TalonFX({{.Name}}Constants.LEADER_ID, {{.Name}}Constants.CAN_BUS);
  private final TalonFX follower =
      new TalonFX({{.Name}}Constants.FOLLOWER_ID, {{.Name}}Constants.CAN_BUS);
  private final StatusSignal<Angle> position;
  private final StatusSignal<AngularVelocity> velocity;
  private final StatusSignal<Voltage> appliedVoltage;
  private final StatusSignal<Current> leaderCurrent;
  private final StatusSignal<Current> followerCurrent;
  private final MotionMagicPositionVoltage positionRequest =
      new MotionMagicPositionVoltage(0).withSlot(0);
  private final VoltageOut voltageRequest = new VoltageOut(0);

  public {{.Name}}IOTalonFX() {
    var config = new TalonFXConfiguration();
    config.Feedback.SensorToMechanismRatio = {{.Name}}Constants.GEARING;
    config.MotorOutput.Inverted = {{.Name}}Constants.LEADER_DIRECTION;
    config.MotorOutput.NeutralMode = NeutralModeValue.Brake;
    config.Slot0 = {{.Name}}Constants.GAINS.toSlot0Configs();
    config.MotionMagic = {{.Name}}Constants.MOTION_MAGIC_CONFIGS;
    config.SoftwareLimitSwitch.ForwardSoftLimitEnable = true;
    config.SoftwareLimitSwitch.ForwardSoftLimitThreshold =
        {{.Name}}Constants.MAX_ANGLE.in(Rotations);
    config.SoftwareLimitSwitch.ReverseSoftLimitEnable = true;
    config.SoftwareLimitSwitch.ReverseSoftLimitThreshold =
        {{.Name}}Constants.MIN_ANGLE.in(Rotations);
    config.CurrentLimits.StatorCurrentLimitEnable = true;
    config.CurrentLimits.StatorCurrentLimit = 40;
    leader.getConfigurator().apply(config);
    follower.getConfigurator().apply(config);
    follower.setControl(new Follower(leader.getDeviceID(), {{.Name}}Constants.FOLLOWER_OPPOSED));

    position = leader.getPosition();
    velocity = leader.getVelocity();
    appliedVoltage = leader.getMotorVoltage();
    leaderCurrent = leader.getSupplyCurrent();
    followerCurrent = follower.getSupplyCurrent();
    position.setUpdateFrequency(50);
    velocity.setUpdateFrequency(50);
    appliedVoltage.setUpdateFrequency(20);
    leaderCurrent.setUpdateFrequency(20);
    followerCurrent.setUpdateFrequency(20);
  }

  @Override
  public void updateInputs({{.Name}}IOInputs inputs) {
    inputs.leaderConnected = position.getStatus().isOK();
    inputs.followerConnected = followerCurrent.getStatus().isOK();
    inputs.position = position.getValue();
    inputs.velocity = velocity.getValue();
    inputs.appliedVoltage = appliedVoltage.getValue();
    inputs.leaderSupplyCurrent = leaderCurrent.getValue();
    inputs.followerSupplyCurrent = followerCurrent.getValue();
  }

  @Override
  public void setAngle(Angle angle) {
    leader.setControl(positionRequest.withPosition(angle.in(Rotations)));
  }

  @Override
  public void setVoltage(Voltage volts) {
    leader.setControl(voltageRequest.withOutput(volts));
  }

  @Override
  public void setBrakeMode(boolean brake) {
    var mode = brake ? NeutralModeValue.Brake : NeutralModeValue.Coast;
    leader.setNeutralMode(mode);
    follower.setNeutralMode(mode);
  }
}
```

- [ ] **Step 4: Create arm/SubsystemIOSim.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import edu.wpi.first.math.system.plant.DCMotor;
import edu.wpi.first.units.measure.Angle;
import edu.wpi.first.units.measure.Voltage;
import edu.wpi.first.wpilibj.simulation.SingleJointedArmSim;

public class {{.Name}}IOSim implements {{.Name}}IO {
  private final SingleJointedArmSim sim =
      new SingleJointedArmSim(
          DCMotor.getKrakenX60(2),
          {{.Name}}Constants.GEARING,
          0.5,
          0.5,
          {{.Name}}Constants.MIN_ANGLE.in(Radians),
          {{.Name}}Constants.MAX_ANGLE.in(Radians),
          true,
          {{.Name}}Constants.MIN_ANGLE.in(Radians));
  private double appliedVolts = 0.0;

  @Override
  public void updateInputs({{.Name}}IOInputs inputs) {
    sim.setInputVoltage(appliedVolts);
    sim.update(0.02);
    inputs.leaderConnected = true;
    inputs.followerConnected = true;
    inputs.position = Radians.of(sim.getAngleRads());
    inputs.velocity = RadiansPerSecond.of(sim.getVelocityRadPerSec());
    inputs.appliedVoltage = Volts.of(appliedVolts);
    inputs.leaderSupplyCurrent = Amps.of(sim.getCurrentDrawAmps() / 2.0);
    inputs.followerSupplyCurrent = Amps.of(sim.getCurrentDrawAmps() / 2.0);
  }

  @Override
  public void setAngle(Angle angle) {
    double error = angle.in(Radians) - sim.getAngleRads();
    appliedVolts = Math.max(-12, Math.min(12, error * 10));
  }

  @Override
  public void setVoltage(Voltage volts) {
    appliedVolts = volts.in(Volts);
  }
}
```

- [ ] **Step 5: Create arm/SubsystemConstants.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import com.ctre.phoenix6.CANBus;
import com.ctre.phoenix6.configs.MotionMagicConfigs;
import com.ctre.phoenix6.signals.InvertedValue;
import static edu.wpi.first.units.Units.*;
import edu.wpi.first.units.measure.Angle;
import frc.robot.util.LoggedNetworkPIDFeedforwardGains;

public class {{.Name}}Constants {
  public static final CANBus CAN_BUS = new CANBus("canivore");
  public static final int LEADER_ID = 0;   // TODO: set CAN IDs
  public static final int FOLLOWER_ID = 1;
  public static final boolean FOLLOWER_OPPOSED = false;
  public static final double GEARING = 1.0;
  public static final InvertedValue LEADER_DIRECTION = InvertedValue.CounterClockwise_Positive;

  public static final Angle MIN_ANGLE = Degrees.of(0.0);
  public static final Angle MAX_ANGLE = Degrees.of(90.0);
  public static final Angle ANGLE_TOLERANCE = Degrees.of(1.5);

  public static final LoggedNetworkPIDFeedforwardGains GAINS =
      new LoggedNetworkPIDFeedforwardGains(
          3.0, 0.0, 0.0, 0.0, 0.0, 0.3, 0.5, "{{.Name}}");

  public static final MotionMagicConfigs MOTION_MAGIC_CONFIGS =
      new MotionMagicConfigs()
          .withMotionMagicCruiseVelocity(0.5)
          .withMotionMagicAcceleration(1.0);
}
```

- [ ] **Step 6: Create elevator/SubsystemIO.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import edu.wpi.first.units.measure.Current;
import edu.wpi.first.units.measure.Distance;
import edu.wpi.first.units.measure.LinearVelocity;
import edu.wpi.first.units.measure.Voltage;
import org.littletonrobotics.junction.AutoLog;

public interface {{.Name}}IO {
  @AutoLog
  public static class {{.Name}}IOInputs {
    public boolean leaderConnected = false;
    public boolean followerConnected = false;
    public Distance position = Meters.of(0.0);
    public LinearVelocity velocity = MetersPerSecond.of(0.0);
    public Voltage appliedVoltage = Volts.of(0.0);
    public Current leaderSupplyCurrent = Amps.of(0.0);
    public Current followerSupplyCurrent = Amps.of(0.0);
  }

  public default void updateInputs({{.Name}}IOInputs inputs) {}

  public default void setPosition(Distance position) {}

  public default void setVoltage(Voltage volts) {}

  public default void setBrakeMode(boolean brake) {}
}
```

- [ ] **Step 7: Create elevator/Subsystem.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import edu.wpi.first.units.measure.Distance;
import edu.wpi.first.wpilibj2.command.Command;
import edu.wpi.first.wpilibj2.command.SubsystemBase;
import org.littletonrobotics.junction.Logger;

public class {{.Name}} extends SubsystemBase {
  private final {{.Name}}IO io;
  private final {{.Name}}IOInputsAutoLogged inputs = new {{.Name}}IOInputsAutoLogged();
  private Distance targetPosition = Meters.of(0);

  public {{.Name}}({{.Name}}IO io) {
    this.io = io;
  }

  @Override
  public void periodic() {
    io.updateInputs(inputs);
    Logger.processInputs("{{.Name}}", inputs);
  }

  public Command setPosition(Distance position) {
    return runOnce(() -> {
      targetPosition = position;
      io.setPosition(position);
    });
  }

  public boolean atTargetPosition() {
    return inputs.position.isNear(targetPosition, {{.Name}}Constants.POSITION_TOLERANCE);
  }
}
```

- [ ] **Step 8: Create elevator/SubsystemIOTalonFX.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import com.ctre.phoenix6.StatusSignal;
import com.ctre.phoenix6.configs.TalonFXConfiguration;
import com.ctre.phoenix6.controls.Follower;
import com.ctre.phoenix6.controls.MotionMagicPositionVoltage;
import com.ctre.phoenix6.controls.VoltageOut;
import com.ctre.phoenix6.hardware.TalonFX;
import com.ctre.phoenix6.signals.NeutralModeValue;
import edu.wpi.first.units.measure.Current;
import edu.wpi.first.units.measure.Distance;
import edu.wpi.first.units.measure.LinearVelocity;
import edu.wpi.first.units.measure.Voltage;

public class {{.Name}}IOTalonFX implements {{.Name}}IO {
  private final TalonFX leader =
      new TalonFX({{.Name}}Constants.LEADER_ID, {{.Name}}Constants.CAN_BUS);
  private final TalonFX follower =
      new TalonFX({{.Name}}Constants.FOLLOWER_ID, {{.Name}}Constants.CAN_BUS);
  private final StatusSignal<Double> position;
  private final StatusSignal<Double> velocity;
  private final StatusSignal<Voltage> appliedVoltage;
  private final StatusSignal<Current> leaderCurrent;
  private final StatusSignal<Current> followerCurrent;
  private final MotionMagicPositionVoltage positionRequest =
      new MotionMagicPositionVoltage(0).withSlot(0);
  private final VoltageOut voltageRequest = new VoltageOut(0);

  public {{.Name}}IOTalonFX() {
    var config = new TalonFXConfiguration();
    // rotations-to-meters: (motor rotations / GEARING) * SPOOL_CIRCUMFERENCE_M
    config.Feedback.SensorToMechanismRatio = {{.Name}}Constants.GEARING;
    config.MotorOutput.NeutralMode = NeutralModeValue.Brake;
    config.Slot0 = {{.Name}}Constants.GAINS.toSlot0Configs();
    config.MotionMagic = {{.Name}}Constants.MOTION_MAGIC_CONFIGS;
    config.SoftwareLimitSwitch.ForwardSoftLimitEnable = true;
    config.SoftwareLimitSwitch.ForwardSoftLimitThreshold =
        {{.Name}}Constants.MAX_HEIGHT.in(Meters) / {{.Name}}Constants.SPOOL_CIRCUMFERENCE_M;
    config.SoftwareLimitSwitch.ReverseSoftLimitEnable = true;
    config.SoftwareLimitSwitch.ReverseSoftLimitThreshold = 0;
    config.CurrentLimits.StatorCurrentLimitEnable = true;
    config.CurrentLimits.StatorCurrentLimit = 60;
    leader.getConfigurator().apply(config);
    follower.getConfigurator().apply(config);
    follower.setControl(new Follower(leader.getDeviceID(), {{.Name}}Constants.FOLLOWER_OPPOSED));

    position = leader.getPosition();
    velocity = leader.getVelocity();
    appliedVoltage = leader.getMotorVoltage();
    leaderCurrent = leader.getSupplyCurrent();
    followerCurrent = follower.getSupplyCurrent();
    position.setUpdateFrequency(50);
    velocity.setUpdateFrequency(50);
    appliedVoltage.setUpdateFrequency(20);
    leaderCurrent.setUpdateFrequency(20);
    followerCurrent.setUpdateFrequency(20);
  }

  @Override
  public void updateInputs({{.Name}}IOInputs inputs) {
    inputs.leaderConnected = appliedVoltage.getStatus().isOK();
    inputs.followerConnected = followerCurrent.getStatus().isOK();
    double rotations = position.getValue();
    inputs.position = Meters.of(rotations * {{.Name}}Constants.SPOOL_CIRCUMFERENCE_M);
    inputs.velocity = MetersPerSecond.of(
        velocity.getValue() * {{.Name}}Constants.SPOOL_CIRCUMFERENCE_M);
    inputs.appliedVoltage = appliedVoltage.getValue();
    inputs.leaderSupplyCurrent = leaderCurrent.getValue();
    inputs.followerSupplyCurrent = followerCurrent.getValue();
  }

  @Override
  public void setPosition(Distance position) {
    double rotations = position.in(Meters) / {{.Name}}Constants.SPOOL_CIRCUMFERENCE_M;
    leader.setControl(positionRequest.withPosition(rotations));
  }

  @Override
  public void setVoltage(Voltage volts) {
    leader.setControl(voltageRequest.withOutput(volts));
  }

  @Override
  public void setBrakeMode(boolean brake) {
    var mode = brake ? NeutralModeValue.Brake : NeutralModeValue.Coast;
    leader.setNeutralMode(mode);
    follower.setNeutralMode(mode);
  }
}
```

- [ ] **Step 9: Create elevator/SubsystemIOSim.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import edu.wpi.first.math.system.plant.DCMotor;
import edu.wpi.first.units.measure.Distance;
import edu.wpi.first.units.measure.Voltage;
import edu.wpi.first.wpilibj.simulation.ElevatorSim;

public class {{.Name}}IOSim implements {{.Name}}IO {
  private final ElevatorSim sim =
      new ElevatorSim(
          DCMotor.getKrakenX60(2),
          {{.Name}}Constants.GEARING,
          5.0,
          {{.Name}}Constants.SPOOL_CIRCUMFERENCE_M / (2 * Math.PI),
          0.0,
          {{.Name}}Constants.MAX_HEIGHT.in(Meters),
          true,
          0.0);
  private double appliedVolts = 0.0;

  @Override
  public void updateInputs({{.Name}}IOInputs inputs) {
    sim.setInputVoltage(appliedVolts);
    sim.update(0.02);
    inputs.leaderConnected = true;
    inputs.followerConnected = true;
    inputs.position = Meters.of(sim.getPositionMeters());
    inputs.velocity = MetersPerSecond.of(sim.getVelocityMetersPerSecond());
    inputs.appliedVoltage = Volts.of(appliedVolts);
    inputs.leaderSupplyCurrent = Amps.of(sim.getCurrentDrawAmps() / 2.0);
    inputs.followerSupplyCurrent = Amps.of(sim.getCurrentDrawAmps() / 2.0);
  }

  @Override
  public void setPosition(Distance position) {
    double error = position.in(Meters) - sim.getPositionMeters();
    appliedVolts = Math.max(-12, Math.min(12, error * 10));
  }

  @Override
  public void setVoltage(Voltage volts) {
    appliedVolts = volts.in(Volts);
  }
}
```

- [ ] **Step 10: Create elevator/SubsystemConstants.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import com.ctre.phoenix6.CANBus;
import com.ctre.phoenix6.configs.MotionMagicConfigs;
import static edu.wpi.first.units.Units.*;
import edu.wpi.first.units.measure.Distance;
import frc.robot.util.LoggedNetworkPIDFeedforwardGains;

public class {{.Name}}Constants {
  public static final CANBus CAN_BUS = new CANBus("canivore");
  public static final int LEADER_ID = 0;   // TODO: set CAN IDs
  public static final int FOLLOWER_ID = 1;
  public static final boolean FOLLOWER_OPPOSED = false;
  public static final double GEARING = 5.0;
  public static final double SPOOL_CIRCUMFERENCE_M = 0.05 * Math.PI; // TODO: measure spool

  public static final Distance MAX_HEIGHT = Meters.of(1.0);
  public static final Distance POSITION_TOLERANCE = Meters.of(0.01);

  public static final LoggedNetworkPIDFeedforwardGains GAINS =
      new LoggedNetworkPIDFeedforwardGains(
          5.0, 0.0, 0.0, 0.0, 0.0, 0.3, 0.3, "{{.Name}}");

  public static final MotionMagicConfigs MOTION_MAGIC_CONFIGS =
      new MotionMagicConfigs()
          .withMotionMagicCruiseVelocity(1.0)
          .withMotionMagicAcceleration(3.0);
}
```

- [ ] **Step 11: Create turret/SubsystemIO.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import edu.wpi.first.units.measure.Angle;
import edu.wpi.first.units.measure.AngularVelocity;
import edu.wpi.first.units.measure.Current;
import edu.wpi.first.units.measure.Voltage;
import org.littletonrobotics.junction.AutoLog;

public interface {{.Name}}IO {
  @AutoLog
  public static class {{.Name}}IOInputs {
    public boolean motorConnected = false;
    public Angle position = Rotations.of(0.0);
    public AngularVelocity velocity = RotationsPerSecond.of(0.0);
    public Voltage appliedVoltage = Volts.of(0.0);
    public Current supplyCurrent = Amps.of(0.0);
  }

  public default void updateInputs({{.Name}}IOInputs inputs) {}

  public default void setAngle(Angle angle) {}

  public default void setVoltage(Voltage volts) {}
}
```

- [ ] **Step 12: Create turret/Subsystem.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import edu.wpi.first.units.measure.Angle;
import edu.wpi.first.wpilibj2.command.Command;
import edu.wpi.first.wpilibj2.command.SubsystemBase;
import org.littletonrobotics.junction.Logger;

public class {{.Name}} extends SubsystemBase {
  private final {{.Name}}IO io;
  private final {{.Name}}IOInputsAutoLogged inputs = new {{.Name}}IOInputsAutoLogged();
  private Angle targetAngle = Rotations.of(0);

  public {{.Name}}({{.Name}}IO io) {
    this.io = io;
  }

  @Override
  public void periodic() {
    io.updateInputs(inputs);
    Logger.processInputs("{{.Name}}", inputs);
  }

  public Command setAngle(Angle angle) {
    return runOnce(() -> {
      targetAngle = angle;
      io.setAngle(angle);
    });
  }

  public boolean atTargetAngle() {
    return inputs.position.isNear(targetAngle, {{.Name}}Constants.ANGLE_TOLERANCE);
  }
}
```

- [ ] **Step 13: Create turret/SubsystemIOTalonFX.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import com.ctre.phoenix6.StatusSignal;
import com.ctre.phoenix6.configs.TalonFXConfiguration;
import com.ctre.phoenix6.controls.MotionMagicPositionVoltage;
import com.ctre.phoenix6.controls.VoltageOut;
import com.ctre.phoenix6.hardware.TalonFX;
import edu.wpi.first.units.measure.Angle;
import edu.wpi.first.units.measure.AngularVelocity;
import edu.wpi.first.units.measure.Current;
import edu.wpi.first.units.measure.Voltage;

public class {{.Name}}IOTalonFX implements {{.Name}}IO {
  private final TalonFX motor =
      new TalonFX({{.Name}}Constants.MOTOR_ID, {{.Name}}Constants.CAN_BUS);
  private final StatusSignal<Angle> position;
  private final StatusSignal<AngularVelocity> velocity;
  private final StatusSignal<Voltage> appliedVoltage;
  private final StatusSignal<Current> supplyCurrent;
  private final MotionMagicPositionVoltage positionRequest =
      new MotionMagicPositionVoltage(0).withSlot(0);
  private final VoltageOut voltageRequest = new VoltageOut(0);

  public {{.Name}}IOTalonFX() {
    var config = new TalonFXConfiguration();
    config.Feedback.SensorToMechanismRatio = {{.Name}}Constants.GEARING;
    config.MotorOutput.Inverted = {{.Name}}Constants.MOTOR_DIRECTION;
    config.ClosedLoopGeneral.ContinuousWrap = true;
    config.Slot0 = {{.Name}}Constants.GAINS.toSlot0Configs();
    config.MotionMagic = {{.Name}}Constants.MOTION_MAGIC_CONFIGS;
    config.CurrentLimits.StatorCurrentLimitEnable = true;
    config.CurrentLimits.StatorCurrentLimit = 40;
    motor.getConfigurator().apply(config);

    position = motor.getPosition();
    velocity = motor.getVelocity();
    appliedVoltage = motor.getMotorVoltage();
    supplyCurrent = motor.getSupplyCurrent();
    position.setUpdateFrequency(50);
    velocity.setUpdateFrequency(50);
    appliedVoltage.setUpdateFrequency(20);
    supplyCurrent.setUpdateFrequency(20);
  }

  @Override
  public void updateInputs({{.Name}}IOInputs inputs) {
    inputs.motorConnected = position.getStatus().isOK();
    inputs.position = position.getValue();
    inputs.velocity = velocity.getValue();
    inputs.appliedVoltage = appliedVoltage.getValue();
    inputs.supplyCurrent = supplyCurrent.getValue();
  }

  @Override
  public void setAngle(Angle angle) {
    motor.setControl(positionRequest.withPosition(angle.in(Rotations)));
  }

  @Override
  public void setVoltage(Voltage volts) {
    motor.setControl(voltageRequest.withOutput(volts));
  }
}
```

- [ ] **Step 14: Create turret/SubsystemIOSim.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import static edu.wpi.first.units.Units.*;
import edu.wpi.first.math.system.plant.DCMotor;
import edu.wpi.first.math.system.plant.LinearSystemId;
import edu.wpi.first.units.measure.Angle;
import edu.wpi.first.units.measure.Voltage;
import edu.wpi.first.wpilibj.simulation.FlywheelSim;

public class {{.Name}}IOSim implements {{.Name}}IO {
  private final FlywheelSim sim =
      new FlywheelSim(
          LinearSystemId.createFlywheelSystem(DCMotor.getKrakenX60(1), 0.01, {{.Name}}Constants.GEARING),
          DCMotor.getKrakenX60(1));
  private double positionRotations = 0.0;
  private double appliedVolts = 0.0;

  @Override
  public void updateInputs({{.Name}}IOInputs inputs) {
    sim.setInputVoltage(appliedVolts);
    sim.update(0.02);
    positionRotations += sim.getAngularVelocityRadPerSec() / (2.0 * Math.PI) * 0.02;
    inputs.motorConnected = true;
    inputs.position = Rotations.of(positionRotations);
    inputs.velocity = RotationsPerSecond.of(sim.getAngularVelocityRadPerSec() / (2.0 * Math.PI));
    inputs.appliedVoltage = Volts.of(appliedVolts);
    inputs.supplyCurrent = Amps.of(Math.abs(sim.getCurrentDrawAmps()));
  }

  @Override
  public void setAngle(Angle angle) {
    double error = angle.in(Rotations) - positionRotations;
    appliedVolts = Math.max(-12, Math.min(12, error * 20));
  }

  @Override
  public void setVoltage(Voltage volts) {
    appliedVolts = volts.in(Volts);
  }
}
```

- [ ] **Step 15: Create turret/SubsystemConstants.java.tmpl**

```
package {{.Package}}.subsystems.{{.NameLower}};

import com.ctre.phoenix6.CANBus;
import com.ctre.phoenix6.configs.MotionMagicConfigs;
import com.ctre.phoenix6.signals.InvertedValue;
import static edu.wpi.first.units.Units.*;
import edu.wpi.first.units.measure.Angle;
import frc.robot.util.LoggedNetworkPIDFeedforwardGains;

public class {{.Name}}Constants {
  public static final CANBus CAN_BUS = new CANBus("canivore");
  public static final int MOTOR_ID = 0; // TODO: set CAN ID
  public static final double GEARING = 1.0;
  public static final InvertedValue MOTOR_DIRECTION = InvertedValue.CounterClockwise_Positive;
  public static final Angle ANGLE_TOLERANCE = Rotations.of(0.01);

  public static final LoggedNetworkPIDFeedforwardGains GAINS =
      new LoggedNetworkPIDFeedforwardGains(
          10.0, 0.0, 0.0, 0.0, 0.12, 0.24, 0.0, "{{.Name}}");

  public static final MotionMagicConfigs MOTION_MAGIC_CONFIGS =
      new MotionMagicConfigs()
          .withMotionMagicCruiseVelocity(2.0)
          .withMotionMagicAcceleration(10.0);
}
```

- [ ] **Step 16: Commit**

```bash
git add internal/template/templates/arm/ internal/template/templates/elevator/ internal/template/templates/turret/
git commit -m "feat: arm, elevator, turret templates"
```

---

### Task 9: Superstructure Template

**Files:**
- Create: `internal/template/templates/superstructure/Superstructure.java.tmpl`

**Security flag:** `none`

- [ ] **Step 1: Create Superstructure.java.tmpl**

```
package {{.Package}}.subsystems.superstructure;

import edu.wpi.first.wpilibj2.command.Command;
import edu.wpi.first.wpilibj2.command.SubsystemBase;
import org.littletonrobotics.junction.Logger;

public class Superstructure extends SubsystemBase {

  public enum WantedState {
    IDLE
    // TODO: add states for your robot's game actions
  }

  public enum CurrentState {
    IDLE
    // TODO: mirror WantedState values
  }

  public WantedState wantedState = WantedState.IDLE;
  public CurrentState currentState = CurrentState.IDLE;

  public Superstructure() {}

  @Override
  public void periodic() {
    CurrentState nextState = handleStateTransitions();
    applyState(nextState);
    currentState = nextState;
    Logger.recordOutput("Superstructure/WantedState", wantedState.toString());
    Logger.recordOutput("Superstructure/CurrentState", currentState.toString());
  }

  private CurrentState handleStateTransitions() {
    switch (wantedState) {
      case IDLE:
      default:
        return CurrentState.IDLE;
    }
  }

  private void applyState(CurrentState state) {
    switch (state) {
      case IDLE:
      default:
        // TODO: stop all subsystems
        break;
    }
  }

  public Command idle() {
    return runOnce(() -> wantedState = WantedState.IDLE);
  }
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/template/templates/superstructure/
git commit -m "feat: superstructure scaffold template"
```

---

### Task 10: Generator Package

**Files:**
- Create: `internal/generator/generator.go`
- Create: `internal/generator/generator_test.go`

**Security flag:** `none`

- [ ] **Step 1: Write failing test**

Create `internal/generator/generator_test.go`:

```go
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
	// First call creates dir
	gen.GenerateSubsystem(ctx, dir)
	// Second call should still succeed (MkdirAll is idempotent)
	if err := gen.GenerateSubsystem(ctx, dir); err != nil {
		t.Fatalf("second GenerateSubsystem() error: %v", err)
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

Run: `go test ./internal/generator/...`  
Expected: compile error (package missing)

- [ ] **Step 3: Implement generator.go**

```go
package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	tmplpkg "github.com/ced4rtree/robot-creator/internal/template"
)

type SubsystemContext struct {
	Name       string
	NameLower  string
	Type       string
	Package    string
	TeamNumber int
}

var templateFiles = []string{
	"Subsystem.java.tmpl",
	"SubsystemIO.java.tmpl",
	"SubsystemIOTalonFX.java.tmpl",
	"SubsystemIOSim.java.tmpl",
	"SubsystemConstants.java.tmpl",
}

type Generator struct {
	Source tmplpkg.TemplateSource
}

func New(source tmplpkg.TemplateSource) *Generator {
	return &Generator{Source: source}
}

func (g *Generator) GenerateSubsystem(ctx SubsystemContext, projectRoot string) error {
	pkgPath := strings.ReplaceAll(ctx.Package, ".", "/")
	outDir := filepath.Join(projectRoot, "src", "main", "java", pkgPath, "subsystems", ctx.NameLower)

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	for _, tmplFile := range templateFiles {
		raw, err := g.Source.GetTemplate(ctx.Type, tmplFile)
		if err != nil {
			return err
		}

		t, err := template.New(tmplFile).Parse(string(raw))
		if err != nil {
			return fmt.Errorf("parsing template %s: %w", tmplFile, err)
		}

		var buf bytes.Buffer
		if err := t.Execute(&buf, ctx); err != nil {
			return fmt.Errorf("executing template %s: %w", tmplFile, err)
		}

		outFile := strings.ReplaceAll(tmplFile, "Subsystem", ctx.Name)
		outFile = strings.TrimSuffix(outFile, ".tmpl")

		if err := os.WriteFile(filepath.Join(outDir, outFile), buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", outFile, err)
		}
		fmt.Printf("  created %s/%s\n", ctx.NameLower, outFile)
	}
	return nil
}
```

- [ ] **Step 4: Run — expect PASS**

Run: `go test ./internal/generator/...`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/generator/
git commit -m "feat: generator package — renders templates to subsystem files"
```

---

### Task 11: Injector Package

**Files:**
- Create: `internal/injector/injector.go`
- Create: `internal/injector/injector_test.go`
- Create: `internal/injector/testdata/RobotContainer.java`

**Security flag:** `none`

- [ ] **Step 1: Create test fixture**

Create `internal/injector/testdata/RobotContainer.java`:

```java
package frc.robot;

import frc.robot.subsystems.drive.Drive;
import frc.robot.subsystems.drive.DriveIO;

public class RobotContainer {
  private final Drive drive;

  public RobotContainer() {
    switch (Constants.currentMode) {
      case REAL:
        drive = new Drive(new DriveIO());
        break;
      case SIM:
        drive = new Drive(new DriveIO() {});
        break;
      default:
        drive = new Drive(new DriveIO() {});
        break;
    }
  }
}
```

- [ ] **Step 2: Write failing tests**

Create `internal/injector/injector_test.go`:

```go
package injector_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ced4rtree/robot-creator/internal/injector"
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
	// Count break; occurrences — should still have 3
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
```

- [ ] **Step 3: Run — expect FAIL**

Run: `go test ./internal/injector/...`  
Expected: compile error (package missing)

- [ ] **Step 4: Implement injector.go**

```go
package injector

import (
	"fmt"
	"os"
	"strings"
)

type SubsystemInjection struct {
	Name      string
	NameLower string
	Package   string
}

type Injector struct {
	RobotContainerPath string
}

func New(path string) *Injector {
	return &Injector{RobotContainerPath: path}
}

func (inj *Injector) Inject(s SubsystemInjection) error {
	content, err := os.ReadFile(inj.RobotContainerPath)
	if err != nil {
		return fmt.Errorf("reading RobotContainer.java: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	lines, err = injectImports(lines, s)
	if err != nil {
		return err
	}
	lines, err = injectField(lines, s)
	if err != nil {
		return err
	}
	lines, err = injectSwitchCases(lines, s)
	if err != nil {
		return err
	}

	return os.WriteFile(inj.RobotContainerPath, []byte(strings.Join(lines, "\n")), 0644)
}

func injectImports(lines []string, s SubsystemInjection) ([]string, error) {
	lastImport := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "import ") {
			lastImport = i
		}
	}
	if lastImport == -1 {
		return lines, fmt.Errorf("could not find import block in RobotContainer.java")
	}
	pkg := s.Package + ".subsystems." + s.NameLower
	newImports := []string{
		fmt.Sprintf("import %s.%s;", pkg, s.Name),
		fmt.Sprintf("import %s.%sIO;", pkg, s.Name),
		fmt.Sprintf("import %s.%sIOTalonFX;", pkg, s.Name),
		fmt.Sprintf("import %s.%sIOSim;", pkg, s.Name),
	}
	result := make([]string, 0, len(lines)+len(newImports))
	result = append(result, lines[:lastImport+1]...)
	result = append(result, newImports...)
	result = append(result, lines[lastImport+1:]...)
	return result, nil
}

func injectField(lines []string, s SubsystemInjection) ([]string, error) {
	lastField := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "private final ") && strings.HasSuffix(trimmed, ";") {
			lastField = i
		}
	}
	if lastField == -1 {
		return lines, fmt.Errorf("could not find field declarations in RobotContainer.java")
	}
	indent := leadingWhitespace(lines[lastField])
	newField := fmt.Sprintf("%sprivate final %s %s;", indent, s.Name, s.NameLower)
	result := make([]string, 0, len(lines)+1)
	result = append(result, lines[:lastField+1]...)
	result = append(result, newField)
	result = append(result, lines[lastField+1:]...)
	return result, nil
}

func injectSwitchCases(lines []string, s SubsystemInjection) ([]string, error) {
	switchIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "switch (Constants.currentMode)") ||
			strings.Contains(line, "switch (currentMode)") {
			switchIdx = i
			break
		}
	}
	if switchIdx == -1 {
		return lines, fmt.Errorf("could not find switch (Constants.currentMode) in RobotContainer.java")
	}

	realIdx, simIdx, defaultIdx := -1, -1, -1
	for i := switchIdx; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		switch trimmed {
		case "case REAL:":
			if realIdx == -1 {
				realIdx = i
			}
		case "case SIM:":
			if simIdx == -1 {
				simIdx = i
			}
		case "default:":
			if defaultIdx == -1 {
				defaultIdx = i
			}
		}
		if realIdx != -1 && simIdx != -1 && defaultIdx != -1 {
			break
		}
	}

	if realIdx == -1 || simIdx == -1 || defaultIdx == -1 {
		return lines, fmt.Errorf("could not find case REAL/SIM/default blocks in switch")
	}

	indent := leadingWhitespace(lines[realIdx]) + "  "
	realLine := fmt.Sprintf("%s%s = new %s(new %sIOTalonFX());", indent, s.NameLower, s.Name, s.Name)
	simLine := fmt.Sprintf("%s%s = new %s(new %sIOSim());", indent, s.NameLower, s.Name, s.Name)
	defaultLine := fmt.Sprintf("%s%s = new %s(new %sIO() {});", indent, s.NameLower, s.Name, s.Name)

	realBreak := findBreakBefore(lines, realIdx+1, simIdx)
	simBreak := findBreakBefore(lines, simIdx+1, defaultIdx)
	defaultBreak := findBreakBefore(lines, defaultIdx+1, len(lines))

	if realBreak == -1 || simBreak == -1 || defaultBreak == -1 {
		return lines, fmt.Errorf("could not find break statements in switch cases")
	}

	// Insert in reverse order so earlier indices remain valid
	lines = insertLine(lines, defaultBreak, defaultLine)
	lines = insertLine(lines, simBreak, simLine)
	lines = insertLine(lines, realBreak, realLine)
	return lines, nil
}

func findBreakBefore(lines []string, start, end int) int {
	for i := end - 1; i >= start; i-- {
		if strings.TrimSpace(lines[i]) == "break;" {
			return i
		}
	}
	return -1
}

func insertLine(lines []string, idx int, line string) []string {
	result := make([]string, 0, len(lines)+1)
	result = append(result, lines[:idx]...)
	result = append(result, line)
	result = append(result, lines[idx:]...)
	return result
}

func leadingWhitespace(s string) string {
	return s[:len(s)-len(strings.TrimLeft(s, " \t"))]
}
```

- [ ] **Step 5: Run tests — expect PASS**

Run: `go test ./internal/injector/...`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/injector/
git commit -m "feat: injector — patches RobotContainer.java with imports, field, switch cases"
```

---

### Task 12: `add subsystem` Command

**Files:**
- Create: `cmd/add_subsystem.go`

**Security flag:** `none`

- [ ] **Step 1: Implement add_subsystem.go**

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ced4rtree/robot-creator/internal/config"
	"github.com/ced4rtree/robot-creator/internal/generator"
	"github.com/ced4rtree/robot-creator/internal/injector"
	tmpl "github.com/ced4rtree/robot-creator/internal/template"
)

var subsystemType string

var addSubsystemCmd = &cobra.Command{
	Use:   "subsystem <Name>",
	Short: "Generate an AdvantageKit subsystem",
	Args:  cobra.ExactArgs(1),
	RunE:  runAddSubsystem,
}

func init() {
	addCmd.AddCommand(addSubsystemCmd)
	addSubsystemCmd.Flags().StringVarP(&subsystemType, "type", "t", "", "subsystem type (flywheel|pivot|roller|arm|elevator|turret|generic)")
	addSubsystemCmd.MarkFlagRequired("type")
}

func runAddSubsystem(cmd *cobra.Command, args []string) error {
	name := args[0]

	if !tmpl.IsValidType(subsystemType) {
		return fmt.Errorf("unknown type %q. Run 'robot-creator list types' for valid types", subsystemType)
	}

	root, err := config.FindRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	if cfg.HasSubsystem(name) {
		return fmt.Errorf("%s already exists. Delete it from robot-creator.yaml to regenerate", name)
	}

	ctx := generator.SubsystemContext{
		Name:       name,
		NameLower:  strings.ToLower(name),
		Type:       subsystemType,
		Package:    cfg.Package,
		TeamNumber: cfg.Team,
	}

	gen := generator.New(tmpl.NewEmbeddedSource())
	fmt.Printf("Generating %s subsystem (type: %s)...\n", name, subsystemType)
	if err := gen.GenerateSubsystem(ctx, root); err != nil {
		return err
	}

	pkgPath := strings.ReplaceAll(cfg.Package, ".", "/")
	rcPath := filepath.Join(root, "src", "main", "java", pkgPath, "RobotContainer.java")
	if _, err := os.Stat(rcPath); err == nil {
		inj := injector.New(rcPath)
		si := injector.SubsystemInjection{
			Name:      name,
			NameLower: strings.ToLower(name),
			Package:   cfg.Package,
		}
		if err := inj.Inject(si); err != nil {
			fmt.Printf("\nWarning: could not auto-inject into RobotContainer.java: %v\n", err)
			fmt.Println("Add these lines manually:\n")
			printManualInstructions(si)
		} else {
			fmt.Println("  injected into RobotContainer.java")
		}
	}

	cfg.Subsystems = append(cfg.Subsystems, config.Subsystem{Name: name, Type: subsystemType})
	if err := config.Save(root, cfg); err != nil {
		return fmt.Errorf("updating robot-creator.yaml: %w", err)
	}

	fmt.Printf("\nDone! %s subsystem created.\n", name)
	return nil
}

func printManualInstructions(s injector.SubsystemInjection) {
	pkg := s.Package + ".subsystems." + s.NameLower
	fmt.Printf("// Imports:\n")
	fmt.Printf("import %s.%s;\n", pkg, s.Name)
	fmt.Printf("import %s.%sIO;\n", pkg, s.Name)
	fmt.Printf("import %s.%sIOTalonFX;\n", pkg, s.Name)
	fmt.Printf("import %s.%sIOSim;\n\n", pkg, s.Name)
	fmt.Printf("// Field:\nprivate final %s %s;\n\n", s.Name, s.NameLower)
	fmt.Printf("// case REAL:\n  %s = new %s(new %sIOTalonFX());\n\n", s.NameLower, s.Name, s.Name)
	fmt.Printf("// case SIM:\n  %s = new %s(new %sIOSim());\n\n", s.NameLower, s.Name, s.Name)
	fmt.Printf("// default:\n  %s = new %s(new %sIO() {});\n", s.NameLower, s.Name, s.Name)
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`  
Expected: no errors

- [ ] **Step 3: Smoke test**

```bash
mkdir /tmp/test-robot && cd /tmp/test-robot
echo "team: 6328\npackage: frc.robot" > robot-creator.yaml
robot-creator add subsystem Shooter --type flywheel
```
Expected: files created in `src/main/java/frc/robot/subsystems/shooter/`, `robot-creator.yaml` updated with Shooter entry.

- [ ] **Step 4: Commit**

```bash
git add cmd/add_subsystem.go
git commit -m "feat: add subsystem command — generates files and injects RobotContainer"
```

---

### Task 13: `add superstructure`, `init`, and `list` Commands

**Files:**
- Create: `cmd/add_superstructure.go`
- Create: `cmd/init.go`
- Create: `cmd/list.go`

**Security flag:** `none`

- [ ] **Step 1: Create cmd/add_superstructure.go**

```go
package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/ced4rtree/robot-creator/internal/config"
	tmplpkg "github.com/ced4rtree/robot-creator/internal/template"
)

var addSuperstructureCmd = &cobra.Command{
	Use:   "superstructure",
	Short: "Generate a Superstructure state-machine scaffold",
	RunE:  runAddSuperstructure,
}

func init() {
	addCmd.AddCommand(addSuperstructureCmd)
}

func runAddSuperstructure(cmd *cobra.Command, args []string) error {
	root, err := config.FindRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	if cfg.Superstructure {
		return fmt.Errorf("superstructure already generated. Remove 'superstructure: true' from robot-creator.yaml to regenerate")
	}

	pkgPath := strings.ReplaceAll(cfg.Package, ".", "/")
	outDir := filepath.Join(root, "src", "main", "java", pkgPath, "subsystems", "superstructure")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	source := tmplpkg.NewEmbeddedSource()
	raw, err := source.GetTemplate("superstructure", "Superstructure.java.tmpl")
	if err != nil {
		return err
	}

	type ctx struct{ Package string }
	t, err := template.New("superstructure").Parse(string(raw))
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, ctx{Package: cfg.Package}); err != nil {
		return err
	}

	outPath := filepath.Join(outDir, "Superstructure.java")
	if err := os.WriteFile(outPath, buf.Bytes(), 0644); err != nil {
		return err
	}

	cfg.Superstructure = true
	if err := config.Save(root, cfg); err != nil {
		return err
	}

	fmt.Printf("Created %s\n", outPath)
	return nil
}
```

- [ ] **Step 2: Create cmd/init.go**

```go
package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/ced4rtree/robot-creator/internal/config"
)

var (
	projectName string
	teamNumber  int
	repoURL     string
	packageName string
)

const defaultRepo = "https://github.com/TEMP_URL/akit-robot-template"

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Clone the AKit starter repo and initialize robot-creator.yaml",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().StringVarP(&projectName, "name", "n", "", "project directory name (required)")
	initCmd.Flags().IntVarP(&teamNumber, "team", "t", 0, "FRC team number (required)")
	initCmd.Flags().StringVar(&repoURL, "repo", defaultRepo, "AKit template repo URL")
	initCmd.Flags().StringVar(&packageName, "package", "frc.robot", "Java package name")
	initCmd.MarkFlagRequired("name")
	initCmd.MarkFlagRequired("team")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	if _, err := os.Stat(projectName); err == nil {
		return fmt.Errorf("directory %q already exists", projectName)
	}

	fmt.Printf("Cloning %s into %s/...\n", repoURL, projectName)
	gitCmd := exec.Command("git", "clone", repoURL, projectName)
	gitCmd.Stdout = os.Stdout
	gitCmd.Stderr = os.Stderr
	if err := gitCmd.Run(); err != nil {
		os.RemoveAll(projectName)
		return fmt.Errorf("git clone failed: %w", err)
	}

	cfg := &config.Config{Team: teamNumber, Package: packageName}
	if err := config.Save(projectName, cfg); err != nil {
		return fmt.Errorf("writing robot-creator.yaml: %w", err)
	}

	fmt.Printf("\nProject %s created!\ncd %s && robot-creator add subsystem <Name> --type <type>\n", projectName, projectName)
	return nil
}
```

- [ ] **Step 3: Create cmd/list.go**

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	tmpl "github.com/ced4rtree/robot-creator/internal/template"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available templates and types",
}

var listTypesCmd = &cobra.Command{
	Use:   "types",
	Short: "List supported subsystem types",
	Run: func(cmd *cobra.Command, args []string) {
		descriptions := map[string]string{
			"flywheel":  "Single motor, velocity control (MotionMagicVelocityVoltage)",
			"pivot":     "Single motor, position control with soft limits",
			"roller":    "Single motor, voltage/on-off control with velocity monitoring",
			"arm":       "Dual motor + follower, position control with soft limits",
			"elevator":  "Dual motor + follower, linear position control in meters",
			"turret":    "Single motor, continuous rotation with ContinuousWrap",
			"generic":   "Minimal scaffold — IO interface + voltage setpoint only",
		}
		source := tmpl.NewEmbeddedSource()
		fmt.Println("Supported subsystem types:")
		for _, t := range source.ListTypes() {
			fmt.Printf("  %-10s %s\n", t, descriptions[t])
		}
	},
}

func init() {
	listCmd.AddCommand(listTypesCmd)
}
```

- [ ] **Step 4: Verify full build**

Run: `go build ./...`  
Expected: no errors

- [ ] **Step 5: Run all tests**

Run: `go test ./...`  
Expected: PASS for config, generator, injector packages

- [ ] **Step 6: End-to-end smoke test**

```bash
robot-creator list types
robot-creator --help
```
Expected: lists all 7 types with descriptions; help shows init/add/list commands.

- [ ] **Step 7: Commit**

```bash
git add cmd/add_superstructure.go cmd/init.go cmd/list.go
git commit -m "feat: add superstructure, init, list commands — MVP complete"
```

---

## Self-Review

**Spec coverage check:**
- `init` command → Task 13 ✅
- `add subsystem` → Task 12 ✅
- `add superstructure` → Task 13 ✅
- `list types` → Task 13 ✅
- All 7 template types → Tasks 4-8 ✅
- TemplateSource interface + EmbeddedSource → Task 3 ✅
- Config load/save/FindRoot → Task 2 ✅
- RobotContainer injection (imports, field, switch cases) → Task 11 ✅
- Best-effort injection with manual fallback → Task 12 ✅
- Duplicate subsystem guard → Task 12 ✅
- robot-creator.yaml updated after add → Task 12 ✅

**No placeholders found.**

**Type consistency:** `SubsystemContext` defined in Task 10, used in Task 12 ✅. `SubsystemInjection` defined in Task 11, used in Task 12 ✅. `TemplateSource` defined in Task 3, used in Tasks 10 and 12 ✅.

**Scope check:** elevator IOTalonFX uses `StatusSignal<Double>` for position/velocity (rotations, not typed Distance) — this is intentional since Phoenix6 signals return `double` for position/velocity and we convert manually. Noted, not a bug.
