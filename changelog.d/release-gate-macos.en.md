- **A release waits for the macOS checks, even where a pull request does not.** Pull
  requests merge without waiting for macOS, which is slow, and run only its Go tests and
  compat-check; the installer, faults, Safari and remote checks on macOS run on `main`
  after each merge, and no version is published until the macOS tests have passed there.
