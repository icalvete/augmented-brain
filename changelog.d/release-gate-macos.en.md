- **A release waits for the macOS checks, even where a pull request does not.** Pull
  requests merge without waiting for macOS, which is slow; CI still runs it on `main`
  after each merge, and no version is published until it has passed there.
