- **Finishing the weekly review can no longer lose captures.** `review_done` used to take
  the processed captures out of the inbox before saving the review, so a review it then
  refused (an empty summary, a token quoted in it) left them gone with nothing saved.
  Now the review is saved first, with the captures it takes out written in it, and only
  then do they leave the inbox. Other writes are all or nothing too: a refused change no
  longer leaves a copy in `archive/`, parking a closed task or changing its steps is
  refused, a capture after a last line with no newline keeps a line of its own, and a
  secret's name written by hand with quotes no longer empties the list.
