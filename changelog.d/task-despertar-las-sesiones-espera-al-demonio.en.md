- **A session stopped by the usage limit comes back at the first try.** In 0.26.0-beta
  the watch resumed it a moment too soon: Claude started a copy instead, the watch
  removed the copy, and the session came back two minutes later, on the next look. Now
  it waits until Claude has really let the session go.
