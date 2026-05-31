package frc.robot;

import frc.robot.subsystems.drive.Drive;
import frc.robot.subsystems.drive.DriveIO;

public class RobotContainer {
  private final Drive drive;

  public RobotContainer() {
    switch (Constants.currentMode) {
      case REAL:
        drive = new Drive(new DriveIO());
        break;
      case SIM:
        drive = new Drive(new DriveIO() {});
        break;
      default:
        drive = new Drive(new DriveIO() {});
        break;
    }
  }
}
