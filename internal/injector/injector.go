package injector

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

type SubsystemInjection struct {
	Name      string
	NameLower string
	Package   string
}

func (s SubsystemInjection) pkg() string {
	return s.Package + ".subsystems." + s.NameLower
}

// Injector edits RobotContainer.java, which owns the mechanisms, and Robot.java, whose
// robotPeriodic() must call each mechanism's periodic() since Commands v3 does not.
type Injector struct {
	RobotContainerPath string
	RobotPath          string
}

func New(robotContainerPath, robotPath string) *Injector {
	return &Injector{RobotContainerPath: robotContainerPath, RobotPath: robotPath}
}

// fieldRe matches an uninitialized final field declaration such as "public final Drive drive;".
var fieldRe = regexp.MustCompile(`^\s*(?:(?:public|private|protected)\s+)?final\s+(\w+)\s+(\w+);\s*$`)

// Inject adds the subsystem's imports, field, and construction in each Constants.currentMode case.
func (inj *Injector) Inject(s SubsystemInjection) error {
	return editFile(inj.RobotContainerPath, func(lines []string) ([]string, error) {
		if hasField(lines, s.Name, s.NameLower) {
			return nil, fmt.Errorf("RobotContainer.java already declares %s %s", s.Name, s.NameLower)
		}
		lines, err := injectImports(lines, []string{
			fmt.Sprintf("import %s.%s;", s.pkg(), s.Name),
			fmt.Sprintf("import %s.%sIO;", s.pkg(), s.Name),
			fmt.Sprintf("import %s.%sIOTalonFX;", s.pkg(), s.Name),
			fmt.Sprintf("import %s.%sIOSim;", s.pkg(), s.Name),
		})
		if err != nil {
			return nil, err
		}
		if lines, err = injectField(lines, s); err != nil {
			return nil, err
		}
		return injectSwitchCases(lines, s)
	})
}

// InjectSuperstructure adds the superstructure's import, field, and construction after the
// currentMode switch, once every mechanism exists.
func (inj *Injector) InjectSuperstructure(pkg string, mechanisms []string) error {
	s := SubsystemInjection{Name: "Superstructure", NameLower: "superstructure", Package: pkg}
	return editFile(inj.RobotContainerPath, func(lines []string) ([]string, error) {
		if hasField(lines, s.Name, s.NameLower) {
			return nil, fmt.Errorf("RobotContainer.java already declares Superstructure superstructure")
		}
		lines, err := injectImports(lines, []string{fmt.Sprintf("import %s.%s;", s.pkg(), s.Name)})
		if err != nil {
			return nil, err
		}
		if lines, err = injectField(lines, s); err != nil {
			return nil, err
		}
		switchIdx, _, _, defaultIdx, err := findSwitch(lines)
		if err != nil {
			return nil, err
		}
		indent := leadingWhitespace(lines[switchIdx])
		for i := defaultIdx + 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "}" && leadingWhitespace(lines[i]) == indent {
				line := fmt.Sprintf("%ssuperstructure = new Superstructure(%s);", indent, strings.Join(mechanisms, ", "))
				return insertLine(lines, i+1, line), nil
			}
		}
		return nil, fmt.Errorf("could not find the end of switch (Constants.currentMode)")
	})
}

// Eject removes the imports, field, and assignments Inject or InjectSuperstructure added.
func (inj *Injector) Eject(s SubsystemInjection) error {
	importPrefix := "import " + s.pkg() + "."
	assignPrefix := fmt.Sprintf("%s = new %s(", s.NameLower, s.Name)
	return editFile(inj.RobotContainerPath, func(lines []string) ([]string, error) {
		result := make([]string, 0, len(lines))
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, importPrefix) ||
				strings.HasPrefix(trimmed, assignPrefix) ||
				isField(line, s.Name, s.NameLower) {
				continue
			}
			result = append(result, line)
		}
		if len(result) == len(lines) {
			return nil, fmt.Errorf("no injected lines found for %s in RobotContainer.java", s.Name)
		}
		return result, nil
	})
}

var periodicRe = regexp.MustCompile(`^(\s*)((?:\w+\.)*)(\w+)\.periodic\(\);\s*$`)

// InjectPeriodic adds "<container>.<field>.periodic();" to Robot.robotPeriodic(), before the
// scheduler runs. Mechanisms go before the superstructure so it sees fresh inputs.
func (inj *Injector) InjectPeriodic(field string) error {
	return editFile(inj.RobotPath, func(lines []string) ([]string, error) {
		start := -1
		for i, line := range lines {
			if strings.Contains(line, "void robotPeriodic(") {
				start = i
				break
			}
		}
		if start == -1 {
			return nil, fmt.Errorf("could not find robotPeriodic() in Robot.java")
		}
		schedIdx := -1
		for i := start; i < len(lines); i++ {
			if strings.Contains(lines[i], "Scheduler.getDefault().run()") {
				schedIdx = i
				break
			}
		}
		if schedIdx == -1 {
			return nil, fmt.Errorf("could not find Scheduler.getDefault().run() in robotPeriodic()")
		}

		prefix := "container."
		insertAt := schedIdx
		indent := leadingWhitespace(lines[schedIdx])
		for i := start; i < schedIdx; i++ {
			m := periodicRe.FindStringSubmatch(lines[i])
			if m == nil {
				continue
			}
			if m[3] == field {
				return nil, fmt.Errorf("Robot.java already calls %s%s.periodic()", m[2], field)
			}
			indent, prefix = m[1], m[2]
			if m[3] == "superstructure" {
				insertAt = i
				break
			}
			insertAt = i + 1
		}
		return insertLine(lines, insertAt, fmt.Sprintf("%s%s%s.periodic();", indent, prefix, field)), nil
	})
}

// EjectPeriodic removes the periodic() call InjectPeriodic added.
func (inj *Injector) EjectPeriodic(field string) error {
	return editFile(inj.RobotPath, func(lines []string) ([]string, error) {
		result := make([]string, 0, len(lines))
		for _, line := range lines {
			if m := periodicRe.FindStringSubmatch(line); m != nil && m[3] == field {
				continue
			}
			result = append(result, line)
		}
		if len(result) == len(lines) {
			return nil, fmt.Errorf("no %s.periodic() call found in Robot.java", field)
		}
		return result, nil
	})
}

func editFile(path string, edit func([]string) ([]string, error)) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	lines, err := edit(strings.Split(string(content), "\n"))
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

func isField(line, typeName, name string) bool {
	m := fieldRe.FindStringSubmatch(line)
	return m != nil && m[1] == typeName && m[2] == name
}

func hasField(lines []string, typeName, name string) bool {
	for _, line := range lines {
		if isField(line, typeName, name) {
			return true
		}
	}
	return false
}

func injectImports(lines []string, newImports []string) ([]string, error) {
	lastImport := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "import ") {
			lastImport = i
		}
	}
	if lastImport == -1 {
		return nil, fmt.Errorf("could not find import block in RobotContainer.java")
	}
	result := make([]string, 0, len(lines)+len(newImports))
	result = append(result, lines[:lastImport+1]...)
	result = append(result, newImports...)
	result = append(result, lines[lastImport+1:]...)
	return result, nil
}

// injectField declares the field public, since Robot.java reaches it through the container.
// It goes after the last mechanism field, ahead of the superstructure.
func injectField(lines []string, s SubsystemInjection) ([]string, error) {
	insertAt, indent := -1, ""
	for i, line := range lines {
		m := fieldRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		indent = leadingWhitespace(line)
		if m[1] == "Superstructure" {
			insertAt = i
			break
		}
		insertAt = i + 1
	}
	if insertAt == -1 {
		return nil, fmt.Errorf("could not find field declarations in RobotContainer.java")
	}
	return insertLine(lines, insertAt, fmt.Sprintf("%spublic final %s %s;", indent, s.Name, s.NameLower)), nil
}

func findSwitch(lines []string) (switchIdx, realIdx, simIdx, defaultIdx int, err error) {
	switchIdx, realIdx, simIdx, defaultIdx = -1, -1, -1, -1
	for i, line := range lines {
		if strings.Contains(line, "switch (Constants.currentMode)") ||
			strings.Contains(line, "switch (currentMode)") {
			switchIdx = i
			break
		}
	}
	if switchIdx == -1 {
		return 0, 0, 0, 0, fmt.Errorf("could not find switch (Constants.currentMode) in RobotContainer.java")
	}
	for i := switchIdx; i < len(lines) && (realIdx == -1 || simIdx == -1 || defaultIdx == -1); i++ {
		switch strings.TrimSpace(lines[i]) {
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
	}
	if realIdx == -1 || simIdx == -1 || defaultIdx == -1 {
		return 0, 0, 0, 0, fmt.Errorf("could not find case REAL/SIM/default blocks in switch")
	}
	return switchIdx, realIdx, simIdx, defaultIdx, nil
}

func injectSwitchCases(lines []string, s SubsystemInjection) ([]string, error) {
	_, realIdx, simIdx, defaultIdx, err := findSwitch(lines)
	if err != nil {
		return nil, err
	}

	indent := leadingWhitespace(lines[realIdx]) + "  "
	realLine := fmt.Sprintf("%s%s = new %s(new %sIOTalonFX());", indent, s.NameLower, s.Name, s.Name)
	simLine := fmt.Sprintf("%s%s = new %s(new %sIOSim());", indent, s.NameLower, s.Name, s.Name)
	defaultLine := fmt.Sprintf("%s%s = new %s(new %sIO() {});", indent, s.NameLower, s.Name, s.Name)

	realBreak := findBreakBefore(lines, realIdx+1, simIdx)
	simBreak := findBreakBefore(lines, simIdx+1, defaultIdx)
	defaultBreak := findBreakAfter(lines, defaultIdx+1)

	if realBreak == -1 || simBreak == -1 || defaultBreak == -1 {
		return nil, fmt.Errorf("could not find break statements in switch cases")
	}

	// Insert in reverse order so earlier indices remain valid
	lines = insertLine(lines, defaultBreak, defaultLine)
	lines = insertLine(lines, simBreak, simLine)
	lines = insertLine(lines, realBreak, realLine)
	return lines, nil
}

// findBreakBefore returns the last "break;" in [start, end), the end of a case followed by another.
func findBreakBefore(lines []string, start, end int) int {
	for i := end - 1; i >= start; i-- {
		if strings.TrimSpace(lines[i]) == "break;" {
			return i
		}
	}
	return -1
}

// findBreakAfter returns the first "break;" from start, the end of the switch's last case.
func findBreakAfter(lines []string, start int) int {
	for i := start; i < len(lines); i++ {
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
