- **A space's name or a folder that could slip into a service is refused.** A name written
  by hand with a line break, a slash, a space or a sign such as `&` no longer loads, and
  the error says how to rename it; `remote install` also refuses a path with a line break,
  or on Windows with `%` or a quote, instead of writing a service that runs something else.
- **Background sessions get only the variables they need.** Claude's background program
  kept everything the terminal that started it had, tokens and test switches included, for
  every later session; now it gets home, `PATH`, language, ssh agent, proxies and Claude's
  own settings.
- **Uninstalling on Windows names every program by its full path**, so a `ping.bat` in the
  folder you ran it from does not run.
