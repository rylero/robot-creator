package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/rylero/robot-creator/internal/config"
	tmplpkg "github.com/rylero/robot-creator/internal/template"
	"github.com/spf13/cobra"
)

var wantedStatesFlag string
var activeStatesFlag string

var addSuperstructureCmd = &cobra.Command{
	Use:   "superstructure",
	Short: "Generate a Superstructure state-machine scaffold",
	RunE:  runAddSuperstructure,
}

func init() {
	addCmd.AddCommand(addSuperstructureCmd)
	addSuperstructureCmd.Flags().StringVar(&wantedStatesFlag, "wanted", "IDLE", "comma-separated WantedState enum values")
	addSuperstructureCmd.Flags().StringVar(&activeStatesFlag, "active", "IDLE", "comma-separated CurrentState enum values")
}

type transitionMapping struct {
	Wanted string
	Active string
}

type superstructureCtx struct {
	Package      string
	WantedStates []string
	ActiveStates []string
	Transitions  []transitionMapping
	DefaultActive string
}

func parseStates(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			result = append(result, strings.ToUpper(s))
		}
	}
	return result
}

func buildTransitions(wanted, active []string) []transitionMapping {
	transitions := make([]transitionMapping, len(wanted))
	for i, w := range wanted {
		a := active[0]
		if i < len(active) {
			a = active[i]
		}
		transitions[i] = transitionMapping{Wanted: w, Active: a}
	}
	return transitions
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

	wanted := parseStates(wantedStatesFlag)
	active := parseStates(activeStatesFlag)
	if len(wanted) == 0 {
		return fmt.Errorf("--wanted must contain at least one state")
	}
	if len(active) == 0 {
		return fmt.Errorf("--active must contain at least one state")
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

	ctx := superstructureCtx{
		Package:       cfg.Package,
		WantedStates:  wanted,
		ActiveStates:  active,
		Transitions:   buildTransitions(wanted, active),
		DefaultActive: active[0],
	}

	funcMap := template.FuncMap{
		"add1": func(i int) int { return i + 1 },
		"lowerCamel": func(s string) string {
			parts := strings.Split(strings.ToLower(s), "_")
			if len(parts) == 1 {
				return parts[0]
			}
			result := parts[0]
			for _, p := range parts[1:] {
				if len(p) > 0 {
					result += strings.ToUpper(p[:1]) + p[1:]
				}
			}
			return result
		},
	}
	t, err := template.New("superstructure").Funcs(funcMap).Parse(string(raw))
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, ctx); err != nil {
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
