- **The remote panel has a door of its own.** With the panel published through
  Tailscale, someone else in your network who had the address could pass for this
  computer by changing one header of the request, and then set up a backup of your
  notes to a repository of theirs. Now what comes through Tailscale arrives at a port of
  its own and is always taken as coming from the network. The link for the phone opens
  only through the network, and the one for this computer's browser only here. Opening
  the panel no longer takes away something you publish yourself with `tailscale serve`
  on port 8443: augmented brain only takes back the address it published.
  `panel stop` no longer ends another process of augmented brain that happens to have
  the process number of a panel that crashed. `panel remote` waits for a slow
  Tailscale instead of starting the panel again (which made the open pages stop
  working).
