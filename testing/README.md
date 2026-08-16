# Test organization

Pantheon is implemented as one white-box Go package under `internal/pantheon/`.
Its package tests live beside that implementation so they can exercise the
durable, unexported safety boundaries without becoming a disconnected package.

Future black-box integration tests may live here. Existing Kernel and Weave
tests were moved to `internal/pantheon/` without deleting or changing them.
