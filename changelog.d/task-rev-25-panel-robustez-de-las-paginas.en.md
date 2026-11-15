- **The panel no longer runs a repository's own filters when it shows your
  repositories.** A folder unpacked from an archive brings its `.git/config`, and a
  filter named there could start a program just by opening Today or Repositories. Your
  own filters, such as git-lfs, still work.
- **A note with an endless quote no longer stops the panel.** A line of a million `>`
  used to crash it; quotes now stop at 16 levels, and the rest shows as text.
- **Long texts are saved.** The panel took at most 64 KiB from a form, less than a
  profile written in Russian or Japanese can hold; now it takes 1 MiB, and beyond that
  it says the text is too long instead of a bare "bad request".
- **A problem report no longer carries your computer's name.** The health report stops
  naming the machine in your Tailscale network, and the report takes out this
  computer's name and any `*.ts.net` name.
- **Typing in the panel counts as using it.** It stops after 30 minutes without use;
  writing a long text without changing page no longer leaves you without a panel to
  save it in.
