package generator_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rylero/robot-creator/v3/internal/generator"
	tmpl "github.com/rylero/robot-creator/v3/internal/template"
)

func subsystemCtx(name, typ string, motors int) generator.SubsystemContext {
	ctx := generator.SubsystemContext{
		Name: name, NameLower: strings.ToLower(name), Type: typ,
		Package: "frc.robot", MotorCount: motors, Aligned: true, LeaderID: 10,
	}
	for i := 2; i <= motors; i++ {
		ctx.Followers = append(ctx.Followers, generator.FollowerInfo{Index: i, DefaultID: 10 + i - 1})
	}
	return ctx
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

func TestGenerateSubsystem_AllTypes(t *testing.T) {
	source := tmpl.NewEmbeddedSource()
	for _, typ := range source.ListTypes() {
		for _, motors := range []int{1, 3} {
			t.Run(fmt.Sprintf("%s/%d", typ, motors), func(t *testing.T) {
				dir := t.TempDir()
				if err := generator.New(source).GenerateSubsystem(subsystemCtx("Thing", typ, motors), dir); err != nil {
					t.Fatalf("GenerateSubsystem() error: %v", err)
				}
				outDir := filepath.Join(dir, "src", "main", "java", "frc", "robot", "subsystems", "thing")
				for _, f := range []string{"Thing.java", "ThingIO.java", "ThingIOTalonFX.java", "ThingIOSim.java", "ThingConstants.java"} {
					s := readFile(t, filepath.Join(outDir, f))
					if strings.Contains(s, "{{") || strings.Contains(s, "<no value>") {
						t.Errorf("%s contains unrendered template syntax", f)
					}
					if !strings.HasPrefix(s, "package frc.robot.subsystems.thing;") {
						t.Errorf("%s missing package declaration", f)
					}
					if strings.Contains(s, "edu.wpi.first") || strings.Contains(s, "wpilibj2") {
						t.Errorf("%s imports pre-2027 WPILib packages", f)
					}
				}

				mech := readFile(t, filepath.Join(outDir, "Thing.java"))
				for _, want := range []string{
					"public class Thing implements Mechanism, AutoCloseable",
					"setDefaultCommand(idle());",
					"public Command idle()",
					"ThingInputsAutoLogged inputs",
				} {
					if !strings.Contains(mech, want) {
						t.Errorf("Thing.java missing %q", want)
					}
				}

				talon := readFile(t, filepath.Join(outDir, "ThingIOTalonFX.java"))
				consts := readFile(t, filepath.Join(outDir, "ThingConstants.java"))
				if !strings.Contains(consts, "MOTOR_ID = 10;") {
					t.Error("ThingConstants.java missing MOTOR_ID = 10")
				}
				if motors == 3 {
					for _, want := range []string{"FOLLOWER_2_ID = 11;", "FOLLOWER_3_ID = 12;", "MotorAlignmentValue.Aligned"} {
						if !strings.Contains(consts, want) {
							t.Errorf("ThingConstants.java missing %q", want)
						}
					}
					if strings.Count(talon, "new Follower(") != 2 {
						t.Error("ThingIOTalonFX.java should configure 2 followers")
					}
					if !strings.Contains(mech, "follower3DisconnectedAlert") {
						t.Error("Thing.java missing follower alert")
					}
				} else if strings.Contains(talon, "Follower") || strings.Contains(consts, "FOLLOWER") {
					t.Error("single-motor mechanism references followers")
				}
			})
		}
	}
}

func TestGenerateSubsystem_OpposedFollowers(t *testing.T) {
	dir := t.TempDir()
	ctx := subsystemCtx("Lift", "elevator", 2)
	ctx.Aligned = false
	if err := generator.New(tmpl.NewEmbeddedSource()).GenerateSubsystem(ctx, dir); err != nil {
		t.Fatal(err)
	}
	s := readFile(t, filepath.Join(dir, "src/main/java/frc/robot/subsystems/lift/LiftConstants.java"))
	if !strings.Contains(s, "MotorAlignmentValue.Opposed;") {
		t.Error("expected Opposed follower alignment")
	}
}

func TestGenerateSubsystem_RefusesExistingDir(t *testing.T) {
	dir := t.TempDir()
	gen := generator.New(tmpl.NewEmbeddedSource())
	ctx := subsystemCtx("Test", "generic", 1)
	if err := gen.GenerateSubsystem(ctx, dir); err != nil {
		t.Fatal(err)
	}
	if err := gen.GenerateSubsystem(ctx, dir); err == nil {
		t.Fatal("expected error when the subsystem directory already exists")
	}
}

func TestEnsureSupportFiles(t *testing.T) {
	dir := t.TempDir()
	javaDir := generator.JavaDir(dir, "frc.robot")

	// An existing helper is kept; one missing the expected method is reported.
	os.MkdirAll(filepath.Join(javaDir, "util"), 0755)
	os.WriteFile(filepath.Join(javaDir, "util", "PhoenixUtil.java"), []byte("class PhoenixUtil {}"), 0644)

	warnings, err := generator.New(tmpl.NewEmbeddedSource()).EnsureSupportFiles("frc.robot", dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "applySafetyLimits") {
		t.Errorf("warnings = %v, want one about applySafetyLimits", warnings)
	}
	if readFile(t, filepath.Join(javaDir, "util", "PhoenixUtil.java")) != "class PhoenixUtil {}" {
		t.Error("existing PhoenixUtil.java was overwritten")
	}
	if s := readFile(t, filepath.Join(javaDir, "util", "Measures.java")); !strings.HasPrefix(s, "package frc.robot.util;") {
		t.Error("Measures.java not generated in frc.robot.util")
	}
	if s := readFile(t, filepath.Join(javaDir, "sim", "VelocityPlant.java")); !strings.Contains(s, "import frc.robot.Constants;") {
		t.Error("VelocityPlant.java not generated with the project package")
	}
}
