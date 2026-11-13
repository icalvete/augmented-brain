- **The Windows installer no longer shows winget's error when it gets Git another way.**
  On a Windows just installed, winget often cannot install Git yet, and the installer
  downloads it from Git for Windows' own page instead. Until now it first showed winget's
  last lines of error, which looked like a failure; now they appear only if the other way
  fails too. The same for the desktop app: winget's output shows only if the app is not
  installed.
