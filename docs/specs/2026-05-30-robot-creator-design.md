# robot-creator — Design Spec
**Date:** 2026-05-30  
**Status:** Approved

---

## Overview

`robot-creator` is a Go CLI tool that scaffolds AKit (AdvantageKit) FRC robot projects and generates subsystem code from templates. It targets the AdvantageKit IO-layer pattern used by teams like 6328.

Primary user: personal tool, with potential team adoption.

---

## Scope

### In scope (MVP)
- `init` — clone AKit starter repo, write `robot-creator.yaml`
- `add subsystem <Name> --type <type>` — generate 5-file AKit subsystem + inject into `RobotContainer.java`
- `add superstructure` — generate minimal `Superstructure.java` scaffold
- `list types` — list supported subsystem types
- Supported types: `flywheel`, `pivot`, `roller`, `arm`, `elevator`, `turret`, `generic`
- Embedded templates (compiled into binary via `embed.FS`)
- `TemplateSource` interface to support local/remote sources in the future

### Non-goals (MVP)
- AI-assisted generation (planned post-MVP)
- Remote template registry / template pull command
- Vanilla WPILib / non-AKit codebases
- Button/trigger binding generation (only instantiation in RobotContainer)
- Drive subsystem generation (lives in the starter repo)
- Non-TalonFX hardware implementations

---

## Architecture

### Stack
- **Language:** Go
- **CLI framework:** [cobra](https://github.com/spf13/cobra)
- **Template engine:** `text/template` (stdlib)
- **Template storage:** `embed.FS` (MVP) behind a `TemplateSource` interface
- **Config:** `robot-creator.yaml` (YAML) at project root

### TemplateSource interface
```go
type TemplateSource interface {
    GetTemplate(subsystemType, fileName string) ([]byte, error)
    ListTypes() []string
}
```

MVP ships `EmbeddedTemplateSource`. Future: `LocalTemplateSource` (reads `~/.robot-creator/templates/`), `RemoteTemplateSource`.

### Directory layout (tool source)
```
robot-creator/
├── cmd/
│   ├── root.go
│   ├── init.go
│   ├── add_subsystem.go
│   ├── add_superstructure.go
│   └── list.go
├── internal/
│   ├── config/        # robot-creator.yaml read/write
│   ├── generator/     # file generation logic
│   ├── injector/      # RobotContainer.java injection
│   └── template/      # TemplateSource interface + EmbeddedTemplateSource
├── templates/
│   ├── flywheel/
│   │   ├── Subsystem.java.tmpl
│   │   ├── SubsystemIO.java.tmpl
│   │   ├── SubsystemIOTalonFX.java.tmpl
│   │   ├── SubsystemIOSim.java.tmpl
│   │   └── SubsystemConstants.java.tmpl
│   ├── pivot/  (same structure)
│   ├── roller/
│   ├── arm/
│   ├── elevator/
│   ├── turret/
│   └── generic/
├── docs/specs/
├── go.mod
└── main.go
```

---

## Commands

### `robot-creator init`
```
robot-creator init --name <ProjectName> --team <TeamNumber> [--repo <url>]
```
1. Clone `--repo` (default: `https://github.com/TEMP_URL/akit-robot-template`) into `./<ProjectName>/`
2. Write `robot-creator.yaml` at project root

### `robot-creator add subsystem <Name> --type <type>`
```
robot-creator add subsystem Shooter --type flywheel
```
1. Read `robot-creator.yaml` — abort if `Name` already registered
2. Render 5 templates into `src/main/java/frc/robot/subsystems/<lowercase-name>/`
3. Inject into `RobotContainer.java` (see Injection section)
4. Append entry to `robot-creator.yaml`

### `robot-creator add superstructure`
```
robot-creator add superstructure
```
1. Generate `src/main/java/frc/robot/subsystems/superstructure/Superstructure.java`
2. Minimal scaffold: `State` enum, `periodic()`, `Logger.processInputs()`

### `robot-creator list types`
Print supported subsystem types with a one-line description each.

---

## Template Variables

All templates receive a `SubsystemContext`:
```go
type SubsystemContext struct {
    Name        string  // e.g. "Shooter"
    NameLower   string  // e.g. "shooter"
    Type        string  // e.g. "flywheel"
    Package     string  // e.g. "frc.robot"
    TeamNumber  int
}
```

---

## Generated Files Per Subsystem

| File | Purpose |
|---|---|
| `{Name}.java` | `SubsystemBase` — periodic, commands, state |
| `{Name}IO.java` | IO interface with `@AutoLog` annotated inputs inner class |
| `{Name}IOTalonFX.java` | Real hardware implementation |
| `{Name}IOSim.java` | WPILib simulation implementation |
| `{Name}Constants.java` | `LoggedTunableNumber` constants |

### IO inputs per type
| Type | Key inputs |
|---|---|
| `flywheel` | `velocityRPS`, `appliedVolts`, `currentAmps`, `motorConnected` |
| `pivot` | `absoluteEncoderPosition`, `relativePosition`, `appliedVolts`, `currentAmps`, `motorConnected` |
| `roller` | `velocityRPS`, `appliedVolts`, `currentAmps`, `motorConnected` |
| `arm` | `absoluteEncoderPosition`, `relativePosition`, `appliedVolts`, `currentAmps[]`, `motorConnected` |
| `elevator` | `positionMeters`, `velocityMetersPerSec`, `appliedVolts`, `currentAmps[]`, `motorConnected` |
| `turret` | `absoluteEncoderPosition`, `relativePosition`, `appliedVolts`, `currentAmps`, `motorConnected` |
| `generic` | `appliedVolts`, `currentAmps`, `motorConnected` |

---

## RobotContainer Injection

Injection is **best-effort**. On any parse failure, print the lines the user needs to add and exit with a non-zero code (no partial writes).

### What gets injected
1. **Import block** — `import frc.robot.subsystems.<name>.<Name>;` etc.
2. **Field declaration** — `private final <Name> <nameLower>;` near other field declarations
3. **Mode switch** — inside the `switch (Constants.currentMode)` block:
```java
case REAL -> <nameLower> = new <Name>(new <Name>IOTalonFX());
case SIM  -> <nameLower> = new <Name>(new <Name>IOSim());
default   -> <nameLower> = new <Name>(new <Name>IO() {});
```

### Detection strategy
- Find the mode switch by scanning for `switch (Constants.currentMode)` or `switch (currentMode)`
- Find the field declaration region by looking for existing `private final` declarations
- Insert using line-based text manipulation (not a Java AST parser)
- Detect injection anchor points with regex; abort cleanly if not found

---

## Project Config (`robot-creator.yaml`)

```yaml
team: 6328
package: frc.robot
subsystems:
  - name: Shooter
    type: flywheel
  - name: Hopper
    type: roller
superstructure: true
```

Written on `init`, updated after each `add` command. Used for:
- Duplicate detection (prevent re-generating a subsystem)
- Inferring package path for file placement

---

## Error Handling

| Scenario | Behavior |
|---|---|
| `robot-creator.yaml` not found | Error: "not in a robot-creator project. Run `init` first." |
| Subsystem already in config | Error: "Shooter already exists. Delete it from robot-creator.yaml to regenerate." |
| RobotContainer injection fails | Print injection snippet, instruct manual add, exit non-zero |
| Unknown `--type` | Error: list valid types |
| Git clone fails (init) | Propagate git error, clean up partial directory |

---

## Testing Strategy

- Unit test template rendering (each type → verify output compiles structurally)
- Unit test RobotContainer injector with fixture files representing common RobotContainer patterns
- Integration test: `add subsystem` on a cloned starter repo, verify files exist and RobotContainer is valid Java
- No mocking of the filesystem — use `t.TempDir()` for test isolation

---

## Known Limitations (Non-Goals Documented)

- **AKit version drift:** Templates target AKit 2025.x. When AKit releases breaking changes, templates need manual updates. Version is documented in `robot-creator.yaml`.
- **RobotContainer layout assumptions:** Injection assumes the standard AKit switch-case constructor pattern. Non-standard layouts get a helpful error + manual instructions.
- **Drive not generated:** Swerve drive is too project-specific; lives in the starter repo.
- **Single motor assumed per type:** Multi-motor configs (e.g. dual-motor elevator) require a `generic` base + manual edits for MVP.

---

## Future / Post-MVP

- AI-assisted generation: mix template fragments + Claude API to fill in PID logic, state machines, etc.
- `LocalTemplateSource`: read templates from `~/.robot-creator/templates/` for team customization
- Remote template registry: `robot-creator template pull <github-url>`
- `robot-creator add superstructure` with subsystem-aware state machine generation
