- **`update` takes its release only from GitHub, and a new program that does not start
  never replaces the one that works.** A setting such as `AB_RELEASE_URL` in the
  environment of a Claude session could send the update somewhere else; now `update`
  ignores it and says so (`--from` is for tests). The installer tries the new program
  before it replaces the old one and puts the old one back if it fails, and it refuses to
  run as root (`sudo`), which left everything belonging to root.
