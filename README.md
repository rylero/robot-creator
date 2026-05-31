# robot-creator

CLI for scaffolding [AdvantageKit](https://github.com/Mechanical-Advantage/AdvantageKit) FRC robot subsystems. Generates the full IO-layer file set and injects wiring into `RobotContainer.java`.

## Requirements

- Go 1.21+
- Git (for `init`)

## Install

```
go install github.com/rylero/robot-creator@latest
```

## Quick Start

```bash
# Clone your AKit template repo and create robot-creator.yaml
robot-creator init --name MyRobot --team 1234

cd MyRobot

# Generate a subsystem
robot-creator add subsystem Shooter --type flywheel

# See what's been added
robot-creator list subsystems

# Remove a subsystem
robot-creator remove subsystem Shooter
```

## Commands

| Command | Description |
|---|---|
| `init --name <dir> --team <number>` | Clone the AKit template repo and initialize the project |
| `add subsystem <Name> --type <type>` | Generate a 5-file AKit subsystem (see flags below) |
| `add superstructure` | Generate a superstructure state machine scaffold |
| `remove subsystem <Name>` | Delete generated files and remove from RobotContainer |
| `list types` | Show available subsystem types |
| `list subsystems` | Show subsystems in the current project |
| `version` | Print the installed version |

### `add subsystem` flags

| Flag | Default | Description |
|---|---|---|
| `--type` | (required) | Subsystem type |
| `--motors` | 2 for arm/elevator, 1 otherwise | Number of TalonFX motors |
| `--aligned` | `true` | Followers mechanically aligned to leader (set `false` if motors face opposite directions) |

### `init` flags

| Flag | Default | Description |
|---|---|---|
| `--name` | (required) | Project directory name |
| `--team` | (required) | FRC team number |
| `--package` | `frc.robot` | Java package name |
| `--repo` | `https://github.com/rylero/akit-robot-template` | AKit template repo to clone |

## Subsystem Types

| Type | Control | Motors |
|---|---|---|
| `flywheel` | MotionMagicVelocityVoltage | 1 |
| `pivot` | MotionMagicVoltage + soft limits | 1 |
| `roller` | VoltageOut with velocity monitoring | 1 |
| `arm` | MotionMagicVoltage + soft limits | configurable (default 2) |
| `elevator` | MotionMagicVoltage + soft limits, meters | configurable (default 2) |
| `turret` | MotionMagicVoltage + ContinuousWrap | 1 |
| `generic` | VoltageOut | 1 |
| `manipulator` | MotionMagicVoltage position control | configurable |

## What Gets Generated

For `robot-creator add subsystem Shooter --type flywheel`:

```
src/main/java/frc/robot/subsystems/shooter/
  Shooter.java            # Subsystem class
  ShooterIO.java          # IO interface + @AutoLog inputs
  ShooterIOTalonFX.java   # TalonFX implementation
  ShooterIOSim.java       # WPILib simulation
  ShooterConstants.java   # CAN IDs, gains, physical constants
```

`RobotContainer.java` is updated automatically with imports, a field declaration, and instantiation in each switch case (`REAL`, `SIM`, `default`). If the file cannot be parsed, the required lines are printed for manual insertion.

## Template Repo

The default template (`rylero/akit-robot-template`) is a swerve + vision baseline using:

- TalonFX swerve (MapleSimSwerve for simulation)
- PhotonVision (2-camera pose estimation)
- RobotIdentity (comp/practice bot switching via RIO serial)
- PathPlanner

Point `--repo` at your own fork to use a customized baseline.

## AlertUtils Wiring

Generated IOTalonFX classes call `AlertUtils.processCriticalAlert` on config failure. Wire the rumble callbacks in `Robot.java` or `RobotContainer`:

```java
AlertUtils.criticalErrorRumbleFunction = () -> controller.setRumble(...);
AlertUtils.stopRumbleFunction = () -> controller.setRumble(...);
```
