- **You can check where a download comes from.** Every file of a release (the archives
  and both installers) now carries a provenance record signed by GitHub, saying which
  workflow built it from which commit (`gh attestation verify`), and `checksums.txt`
  lists the installers too. The same commit builds the same bytes anywhere. A release
  is made from the commit of `main` that set its version, never from changes merged
  after it, and only once every required check has passed there; it is built in a job
  that cannot write to the repository. The guide to installing says how to check a
  download.
