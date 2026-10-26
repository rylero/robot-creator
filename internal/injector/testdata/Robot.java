package frc.robot;

import frc.robot.Constants.Mode;
import org.ironmaple.simulation.SimulatedArena;
import org.ironmaple.simulation.seasonspecific.rebuilt2026.Arena2026Rebuilt;
import org.littletonrobotics.junction.AutoLogOutputManager;
import org.littletonrobotics.junction.LogFileUtil;
import org.littletonrobotics.junction.Logger;
import org.littletonrobotics.junction.networktables.NT4Publisher;
import org.littletonrobotics.junction.wpilog.WPILOGReader;
import org.littletonrobotics.junction.wpilog.WPILOGWriter;
import org.wpilib.command3.Scheduler;
import org.wpilib.framework.OpModeRobot;
import org.wpilib.system.RobotController;

/**
 * OpMode robot for the AdvantageKit Phoenix swerve template.
 *
 * <p>AdvantageKit does not yet ship an {@code OpModeRobot} base class. Logging still runs through
 * {@link Logger.AdvancedHooks}, which exists for custom robot bases. Driver Station OpModes in this
 * package and its subpackages are registered automatically.
 */
public class Robot extends OpModeRobot {
  public final RobotContainer container;

  public Robot() {
    super(Constants.ROBOT_PERIOD);

    // Record metadata
    Logger.recordMetadata("ProjectName", BuildConstants.MAVEN_NAME);
    Logger.recordMetadata("BuildDate", BuildConstants.BUILD_DATE);
    Logger.recordMetadata("GitSHA", BuildConstants.GIT_SHA);
    Logger.recordMetadata("GitDate", BuildConstants.GIT_DATE);
    Logger.recordMetadata("GitBranch", BuildConstants.GIT_BRANCH);
    Logger.recordMetadata(
        "GitDirty",
        switch (BuildConstants.DIRTY) {
          case 0 -> "All changes committed";
          case 1 -> "Uncommitted changes";
          default -> "Unknown";
        });

    // Set up data receivers & replay source
    switch (Constants.currentMode) {
      case REAL:
        // Running on a real robot, log to a USB stick ("/U/logs")
        Logger.addDataReceiver(new WPILOGWriter());
        Logger.addDataReceiver(new NT4Publisher());
        break;

      case SIM:
        // Running a physics simulator, log to NT
        Logger.addDataReceiver(new NT4Publisher());
        break;

      case REPLAY:
        // Replaying a log, set up replay source
        String logPath = LogFileUtil.findReplayLog();
        Logger.setReplaySource(new WPILOGReader(logPath));
        Logger.addDataReceiver(new WPILOGWriter(LogFileUtil.addPathSuffix(logPath, "_sim")));
        break;
    }

    // Advantage Kit by default does not work with OPModeRobot, as it wants us to use LoggedRobot
    Logger.AdvancedHooks.disableRobotBaseCheck();
    Logger.start();

    if (Constants.currentMode == Mode.SIM) {
      SimulatedArena.overrideInstance(new Arena2026Rebuilt(false));
    }

    container = new RobotContainer();

    AutoLogOutputManager.addObject(this);
  }

  @Override
  public void robotPeriodic() {
    long beforeStart = RobotController.getMonotonicTime();
    Logger.AdvancedHooks.invokePeriodicBeforeUser();
    long userStart = RobotController.getMonotonicTime();

    // Update mechanism inputs and odometry first so commands see this loop's data. Vision runs
    // after drive so its measurements land on the newest odometry.
    container.drive.periodic();
    container.vision.periodic();
    container.shooter.periodic();
    container.hopper.periodic();
    container.intake.periodic();
    container.superstructure.periodic();

    // Runs the Scheduler. This is responsible for polling buttons, queuing default commands, and
    // running scheduled commands until they yield.
    Scheduler.getDefault().run();

    container.periodic();
    container.updateSimulation();

    long userEnd = RobotController.getMonotonicTime();
    Logger.AdvancedHooks.invokePeriodicAfterUser(userEnd - userStart, userStart - beforeStart);
  }
}
