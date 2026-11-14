- **Claude never gets an address that opens the panel.** Run by Claude with no browser
  to open (a session in the background, a computer with no screen), `augmented-brain
  panel` used to print the address; now it asks you to run it in a terminal of your own,
  and so does `augmented-brain panel remote`. On Linux the browser no longer gets the
  address as an argument, which every user of the computer can read, but a small file
  only you can read (not with a snap or flatpak browser, which cannot read it).
