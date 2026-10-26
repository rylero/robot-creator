package frc.robot;

import static frc.robot.subsystems.vision.VisionConstants.*;

import frc.robot.commands.DriveCommands;
import frc.robot.subsystems.drive.Drive;
import frc.robot.subsystems.drive.GyroIO;
import frc.robot.subsystems.drive.GyroIOPigeon2;
import frc.robot.subsystems.drive.GyroIOSim;
import frc.robot.subsystems.drive.MapleSimSwerve;
import frc.robot.subsystems.drive.ModuleIO;
import frc.robot.subsystems.drive.ModuleIOSim;
import frc.robot.subsystems.drive.ModuleIOTalonFX;
import frc.robot.subsystems.hopper.Hopper;
import frc.robot.subsystems.hopper.HopperIO;
import frc.robot.subsystems.hopper.HopperIOSim;
import frc.robot.subsystems.hopper.HopperIOTalonFX;
import frc.robot.subsystems.intake.Intake;
import frc.robot.subsystems.intake.IntakeIO;
import frc.robot.subsystems.intake.IntakeIOSim;
import frc.robot.subsystems.intake.IntakeIOTalonFX;
import frc.robot.subsystems.shooter.Shooter;
import frc.robot.subsystems.shooter.ShooterIO;
import frc.robot.subsystems.shooter.ShooterIOSim;
import frc.robot.subsystems.shooter.ShooterIOTalonFX;
import frc.robot.subsystems.superstructure.Superstructure;
import frc.robot.subsystems.superstructure.Superstructure.Wanted;
import frc.robot.subsystems.vision.Vision;
import frc.robot.subsystems.vision.VisionIO;
import frc.robot.subsystems.vision.VisionIOPhotonVision;
import frc.robot.subsystems.vision.VisionIOPhotonVisionSim;
import frc.robot.util.RobotIdentity;
import org.ironmaple.simulation.SimulatedArena;
import org.ironmaple.simulation.drivesims.SwerveDriveSimulation;
import org.littletonrobotics.junction.Logger;
import org.wpilib.command3.Command;
import org.wpilib.command3.button.CommandXboxController;
import org.wpilib.math.geometry.Pose2d;
import org.wpilib.math.geometry.Rotation2d;

/**
 * Owns the robot's mechanisms, controllers, and global button bindings. OpModes reach these through
 * {@link Robot#container}.
 */
public class RobotContainer {
  public final Drive drive;
  public final Vision vision;
  public final Shooter shooter;
  public final Hopper hopper;
  public final Intake intake;
  public final Superstructure superstructure;
  public final CommandXboxController controller = new CommandXboxController(0);
  private SwerveDriveSimulation swerveDriveSimulation;

  public RobotContainer() {
    switch (Constants.currentMode) {
      case REAL:
        drive =
            new Drive(
                new GyroIOPigeon2(),
                new ModuleIOTalonFX(RobotIdentity.getTunerConstants().FrontLeft),
                new ModuleIOTalonFX(RobotIdentity.getTunerConstants().FrontRight),
                new ModuleIOTalonFX(RobotIdentity.getTunerConstants().BackLeft),
                new ModuleIOTalonFX(RobotIdentity.getTunerConstants().BackRight));
        vision =
            new Vision(
                drive::addVisionMeasurement,
                drive::getPose,
                new VisionIOPhotonVision(camera0Name, robotToCamera0),
                new VisionIOPhotonVision(camera1Name, robotToCamera1));
        shooter = new Shooter(new ShooterIOTalonFX());
        hopper = new Hopper(new HopperIOTalonFX());
        intake = new Intake(new IntakeIOTalonFX());
        break;

      case SIM:
        swerveDriveSimulation =
            MapleSimSwerve.createSimulationDrive(RobotIdentity.getTunerConstants());
        drive =
            new Drive(
                new GyroIOSim(swerveDriveSimulation.getGyroSimulation()),
                new ModuleIOSim(swerveDriveSimulation.getModules()[0]),
                new ModuleIOSim(swerveDriveSimulation.getModules()[1]),
                new ModuleIOSim(swerveDriveSimulation.getModules()[2]),
                new ModuleIOSim(swerveDriveSimulation.getModules()[3]),
                swerveDriveSimulation);
        vision =
            new Vision(
                drive::addVisionMeasurement,
                drive::getPose,
                new VisionIOPhotonVisionSim(
                    camera0Name, robotToCamera0, swerveDriveSimulation::getSimulatedDriveTrainPose),
                new VisionIOPhotonVisionSim(
                    camera1Name,
                    robotToCamera1,
                    swerveDriveSimulation::getSimulatedDriveTrainPose));
        shooter = new Shooter(new ShooterIOSim());
        hopper = new Hopper(new HopperIOSim());
        intake = new Intake(new IntakeIOSim());
        break;

      default:
        drive =
            new Drive(
                new GyroIO() {},
                new ModuleIO() {},
                new ModuleIO() {},
                new ModuleIO() {},
                new ModuleIO() {});
        vision =
            new Vision(
                drive::addVisionMeasurement, drive::getPose, new VisionIO() {}, new VisionIO() {});
        shooter = new Shooter(new ShooterIO() {});
        hopper = new Hopper(new HopperIO() {});
        intake = new Intake(new IntakeIO() {});
        break;
    }

    superstructure = new Superstructure(drive, shooter, hopper, intake);
    configureButtonBindings();
  }

  /**
   * Bindings made here, before any OpMode is selected, are global. Bindings made inside an OpMode
   * only live as long as that OpMode.
   */
  private void configureButtonBindings() {
    controller
        .a()
        .whileTrue(
            DriveCommands.joystickDriveAtAngle(
                drive,
                () -> -controller.getController().getLeftY(),
                () -> -controller.getController().getLeftX(),
                () -> Rotation2d.ZERO));

    controller.x().onTrue(drive.run(coroutine -> drive.stopWithX()).named("Stop With X"));

    controller
        .b()
        .onTrue(
            drive
                .run(
                    coroutine ->
                        drive.setPose(
                            new Pose2d(drive.getPose().getTranslation(), Rotation2d.ZERO)))
                .named("Reset Heading"));

    controller
        .rightTrigger()
        .whileTrue(
            Command.parallel(
                    DriveCommands.joystickDriveAtAngle(
                        drive,
                        () -> -controller.getController().getLeftY(),
                        () -> -controller.getController().getLeftX(),
                        () -> superstructure.getShotSetpoint().robotPose().getRotation()),
                    superstructure.want(Wanted.SHOOT_INTAKE))
                .named("ShootIntake"))
        .onFalse(superstructure.want(Wanted.IDLE));

    controller
        .rightBumper()
        .whileTrue(
            Command.parallel(
                    DriveCommands.joystickDriveAtAngle(
                        drive,
                        () -> -controller.getController().getLeftY(),
                        () -> -controller.getController().getLeftX(),
                        () -> superstructure.getShotSetpoint().robotPose().getRotation()),
                    superstructure.want(Wanted.SHOOTING))
                .named("Shoot"))
        .onFalse(superstructure.want(Wanted.IDLE));

    controller
        .leftBumper()
        .onTrue(
            superstructure.want(
                () ->
                    superstructure.getWanted() == Wanted.INTAKING ? Wanted.IDLE : Wanted.INTAKING));
    controller
        .leftTrigger()
        .whileTrue(superstructure.want(Wanted.SPIT))
        .onFalse(superstructure.want(Wanted.IDLE));
    controller
        .y()
        .whileTrue(superstructure.want(Wanted.REVERSE))
        .onFalse(superstructure.want(Wanted.IDLE));
    controller
        .dpadLeft()
        .whileTrue(superstructure.want(Wanted.PIT_SHOOT))
        .onFalse(superstructure.want(Wanted.IDLE));
  }

  public void periodic() {}

  /** Steps MapleSim and logs the simulated field pose. */
  public void updateSimulation() {
    if (swerveDriveSimulation == null) {
      return;
    }
    SimulatedArena.getInstance().simulationPeriodic();
    Logger.recordOutput(
        "FieldSimulation/RobotPosition", swerveDriveSimulation.getSimulatedDriveTrainPose());
  }
}
