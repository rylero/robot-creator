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
