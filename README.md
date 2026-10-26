# robot-creator

CLI for scaffolding [AdvantageKit](https://github.com/Mechanical-Advantage/AdvantageKit) FRC mechanisms on **WPILib 2027 Commands v3**. Generates the full IO-layer file set and wires it into `RobotContainer.java` and `Robot.java`.

> Looking for Commands v2 (`SubsystemBase`, `edu.wpi.first.*`)? Use the v0.1.x line:
> `go install github.com/rylero/robot-creator@v0.1.8`

## Requirements

- Go 1.25+
- Git (for `init`)
- A robot project on WPILib 2027 with the Commands v3 vendordep, AdvantageKit, and Phoenix 6, structured like [6238/2026RobotSystemcore](https://github.com/6238/2026RobotSystemcore): an `OpModeRobot` whose `robotPeriodic()` calls each mechanism's `periodic()` and then `Scheduler.getDefault().run()`

## Install

```
go install github.com/rylero/robot-creator/v3@latest
```

## Quick Start

```bash
# Clone a Commands v3 robot repo and create robot-creator.yaml
robot-creator init --name MyRobot --team 1234

# ...or adopt an existing Commands v3 project
cd MyRobot && robot-creator init --no-clone --team 1234

# Generate mechanisms
robot-creator add subsystem Shooter --type flywheel --motors 2 --id 20
robot-creator add subsystem Arm --type arm --id 30
robot-creator add subsystem Wrist --type manipulator

robot-creator list subsystems
robot-creator remove subsystem Shooter
```

## Commands

| Command | Description |
|---|---|
| `init --name <dir> --team <number>` | Clone a Commands v3 robot repo and initialize the project |
| `init --no-clone --team <number>` | Initialize `robot-creator.yaml` in the current directory without cloning |
| `add subsystem <Name> --type <type>` | Generate a 5-file AdvantageKit mechanism (see flags below) |
| `remove subsystem <Name>` | Delete generated files and unwire them |
| `export templates [dir]` | Copy built-in templates to a local directory for customization |
| `list types` | Show available subsystem types |
| `list subsystems` | Show subsystems in the current project |
| `version` | Print the installed version |

### `add subsystem` flags

| Flag | Default | Description |
|---|---|---|
| `--type` | (required) | Subsystem type |
| `--motors` | 2 for arm/elevator, 1 otherwise | Number of TalonFX motors; every type supports followers |
| `--aligned` | `true` | Followers mechanically aligned to the leader; pass `--aligned=false` if they face opposite directions |
| `--id` | `0` | Leader CAN ID; followers get consecutive IDs (`--id 5` → leader=5, follower2=6, follower3=7) |

### `init` flags

| Flag | Default | Description |
|---|---|---|
| `--name` | (required unless `--no-clone`) | Project directory name |
| `--team` | (required) | FRC team number |
| `--package` | `frc.robot` | Java package name |
| `--repo` | `https://github.com/6238/2026RobotSystemcore` | Commands v3 robot repo to clone |
| `--no-clone` | `false` | Skip git clone; write `robot-creator.yaml` into the current directory |

`robot-creator.yaml` carries `version: 3`. The CLI refuses projects without it so Commands v3 code never lands in a Commands v2 robot.

## Subsystem Types

| Type | Control | Commands |
|---|---|---|
| `flywheel` | `VelocityVoltage` | `spin()`, `spin(velocity)`, `atSpeed()` |
| `pivot` | `MotionMagicVoltage`, `Arm_Cosine` gravity, soft limits | `hold(angle)`, `moveTo(angle)`, `atSetpoint()` |
| `arm` | Same as pivot; 2 motors by default | `hold(angle)`, `moveTo(angle)`, `atSetpoint()` |
| `elevator` | `MotionMagicVoltage` in meters, `Elevator_Static` gravity, soft limits; 2 motors by default | `hold(height)`, `moveTo(height)`, `atSetpoint()` |
| `turret` | `MotionMagicVoltage` with `ContinuousWrap` | `hold(angle)`, `moveTo(angle)`, `atSetpoint()` |
| `manipulator` | `MotionMagicVoltage` in rotations, soft limits | `hold(position)`, `moveTo(position)`, `atSetpoint()` |
| `roller` | `VoltageOut` with debounced stall detection | `forward()`, `reverse()`, `runVoltage(v)`, `isStalled()` |
| `generic` | `VoltageOut` | `runVoltage(v)` |

Every mechanism `implements Mechanism, AutoCloseable` and sets `idle()` (0 V) as its default command. `hold(...)` runs until interrupted; `moveTo(...)` finishes once within tolerance, after which the default command takes over, so use `hold` when a mechanism must stay put. `hold`, `spin`, and `runVoltage` also accept a `Supplier`.

### Lead/follower failover

Mechanisms with followers (`--motors 2` or more) keep running if the leading motor drops off the CAN bus:

- When the leader has been disconnected for `CONNECTION_DEBOUNCE_SECONDS` (0.5 s), the first connected motor takes over. The others re-follow it, and the last request (position, velocity or voltage) is re-sent to it.
- Leadership is sticky: when the original leader comes back it follows the new leader rather than taking over again.
- Opposed followers get the opposite `Inverted` config, so every motor reads and accepts setpoints in the same direction. Position control and soft limits therefore stay correct whichever motor leads.
- The inputs log `leader` (1 = configured leader, n = follower n), and the mechanism raises a `failover` alert while a follower is leading.

Single-motor mechanisms don't get any of this code.

Simulation uses WPILib plants: `SingleJointedArmSim` for arm/pivot, `ElevatorSim` for elevator, and the project's `VelocityPlant` for everything else.

## What Gets Generated

For `robot-creator add subsystem Shooter --type flywheel`:

```
src/main/java/frc/robot/subsystems/shooter/
  Shooter.java            # Mechanism with commands and disconnect alerts
  ShooterIO.java          # IO interface + @AutoLog ShooterInputs
  ShooterIOTalonFX.java   # TalonFX implementation (StatusSignalCollection)
  ShooterIOSim.java       # WPILib simulation
  ShooterConstants.java   # CAN IDs, limits, TalonFXConfiguration builder
```

Commands v3 does not call mechanism `periodic()` for you, so the wiring touches two files:

- **`RobotContainer.java`**: imports, a `public final Shooter shooter;` field, and construction in each `Constants.currentMode` case (`REAL`, `SIM`, `default`).
- **`Robot.java`**: `container.shooter.periodic();` in `robotPeriodic()`, after the last existing `periodic()` call and before `Scheduler.getDefault().run()`.

If either file can't be parsed, the lines to add are printed instead. An existing subsystem directory is never overwritten.

The generated code also needs `util/PhoenixUtil`, `util/Measures`, and `sim/VelocityPlant`. Any that are missing from the project are created; existing ones are left alone.

## Custom Templates

```bash
robot-creator export templates           # exports to ./templates/
robot-creator export templates my-tmpls  # exports to ./my-tmpls/
```

After export, `robot-creator.yaml` gets `templates_dir: templates`, and `add subsystem` uses your local `.tmpl` files (Go template syntax). Remove that line to go back to the built-ins.
