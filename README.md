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

# Generate subsystems
robot-creator add subsystem Shooter --type flywheel
robot-creator add subsystem Arm --type arm --motors 3
robot-creator add subsystem Wrist --type manipulator

# Generate a superstructure state machine
robot-creator add superstructure --wanted IDLE,INTAKE,SHOOT --active IDLE,INTAKING,SHOOTING

# See what's been added
robot-creator list subsystems

# Remove a subsystem
robot-creator remove subsystem Shooter
```

## Commands

| Command | Description |
|---|---|
| `init --name <dir> --team <number>` | Clone the AKit template repo and initialize the project |
| `init --no-clone --team <number>` | Initialize `robot-creator.yaml` in the current directory without cloning |
| `add subsystem <Name> --type <type>` | Generate a 5-file AKit subsystem (see flags below) |
| `add superstructure` | Generate a superstructure state machine (see flags below) |
| `remove subsystem <Name>` | Delete generated files and remove from RobotContainer |
| `remove superstructure` | Delete the superstructure directory and clear it from config |
| `export templates [dir]` | Copy built-in templates to a local directory for customization |
| `list types` | Show available subsystem types |
| `list subsystems` | Show subsystems in the current project |
| `version` | Print the installed version |

### `add subsystem` flags

| Flag | Default | Description |
|---|---|---|
| `--type` | (required) | Subsystem type |
| `--motors` | 2 for arm/elevator, 1 otherwise | Number of TalonFX motors |
| `--aligned` | `true` | Followers mechanically aligned to leader; pass `--aligned=false` if motors face opposite directions |
| `--id` | `0` | Leader/motor CAN ID; followers get consecutive IDs (`--id 5` → leader=5, follower2=6, follower3=7) |

### `add superstructure` flags

| Flag | Default | Description |
|---|---|---|
| `--wanted` | `IDLE` | Comma-separated `WantedState` enum values |
| `--active` | `IDLE` | Comma-separated `CurrentState` enum values |

States are mapped by index (`wanted[i]` → `active[i]`). If the lists have different lengths, extra wanted states fall back to `active[0]`. State names are uppercased automatically.

### `init` flags

| Flag | Default | Description |
|---|---|---|
| `--name` | (required unless `--no-clone`) | Project directory name |
| `--team` | (required) | FRC team number |
| `--package` | `frc.robot` | Java package name |
| `--repo` | `https://github.com/rylero/akit-robot-template` | AKit template repo to clone |
| `--no-clone` | `false` | Skip git clone; write `robot-creator.yaml` into the current directory |

## Subsystem Types

| Type | Control | Motors |
|---|---|---|
| `flywheel` | MotionMagicVelocityVoltage | 1 |
| `pivot` | MotionMagicVoltage + soft limits | 1 |
| `roller` | VoltageOut with velocity monitoring | 1 |
| `arm` | MotionMagicVoltage + soft limits (angle) | configurable (default 2) |
| `elevator` | MotionMagicVoltage + soft limits (meters) | configurable (default 2) |
| `turret` | MotionMagicVoltage + ContinuousWrap | 1 |
| `generic` | VoltageOut | 1 |
| `manipulator` | MotionMagicVoltage position control (rotations) | configurable (default 1) |

`arm` and `elevator` use `SingleJointedArmSim`/`ElevatorSim` for physics-accurate simulation. `manipulator` uses a simpler `FlywheelSim` + velocity integration — suitable for wrists, claws, or any position-controlled mechanism where arm inertia is unknown.

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

For `robot-creator add superstructure --wanted IDLE,INTAKE,SHOOT --active IDLE,INTAKING,SHOOTING`:

```
src/main/java/frc/robot/subsystems/superstructure/
  Superstructure.java     # State machine with WantedState/CurrentState enums,
                          # handleStateTransitions(), applyState(), and command methods
```

## Multi-Motor Example

```bash
# 3-motor arm, all aligned
robot-creator add subsystem Arm --type arm --motors 3

# 2-motor elevator with opposed followers (motors face each other)
robot-creator add subsystem Elevator --type elevator --motors 2 --aligned=false
```

Generated code uses `follower2`, `follower3`... naming. Each follower gets its own CAN ID constant (`FOLLOWER_2_ID`, `FOLLOWER_3_ID`...) and supply current signal.

## Custom Templates

Export the built-in templates to edit them locally:

```bash
robot-creator export templates           # exports to ./templates/
robot-creator export templates my-tmpls  # exports to ./my-tmpls/
```

After export, `robot-creator.yaml` is updated with `templates_dir: templates`. All subsequent `add subsystem` and `add superstructure` calls will use your local `.tmpl` files instead of the built-ins. Edit any `.java.tmpl` file — Go template syntax applies.

To revert to built-ins, remove the `templates_dir` line from `robot-creator.yaml`.

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
