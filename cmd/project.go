package cmd

import (
	"os"
	"path/filepath"

	"github.com/rylero/robot-creator/v3/internal/config"
	"github.com/rylero/robot-creator/v3/internal/generator"
	"github.com/rylero/robot-creator/v3/internal/injector"
	tmpl "github.com/rylero/robot-creator/v3/internal/template"
)

func loadProject() (string, *config.Config, error) {
	root, err := config.FindRoot()
	if err != nil {
		return "", nil, err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return "", nil, err
	}
	return root, cfg, nil
}

func templateSource(root string, cfg *config.Config) tmpl.TemplateSource {
	if cfg.TemplatesDir != "" {
		return tmpl.NewLocalSource(filepath.Join(root, cfg.TemplatesDir))
	}
	return tmpl.NewEmbeddedSource()
}

// projectInjector returns an injector for RobotContainer.java and Robot.java, plus whether each
// file exists. Generation still succeeds without them; the wiring is printed instead.
func projectInjector(root string, cfg *config.Config) (inj *injector.Injector, hasContainer, hasRobot bool) {
	dir := generator.JavaDir(root, cfg.Package)
	inj = injector.New(filepath.Join(dir, "RobotContainer.java"), filepath.Join(dir, "Robot.java"))
	_, err := os.Stat(inj.RobotContainerPath)
	hasContainer = err == nil
	_, err = os.Stat(inj.RobotPath)
	hasRobot = err == nil
	return inj, hasContainer, hasRobot
}
