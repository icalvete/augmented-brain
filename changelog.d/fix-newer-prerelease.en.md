- **`update` now sees a release candidate's next step.** It compared only the numbers,
  so after installing 1.0.0-rc.1 it would never have offered 1.0.0-rc.2 or 1.0.0: same
  numbers. Now a release is newer than its candidates, and rc.2 is newer than rc.1.
