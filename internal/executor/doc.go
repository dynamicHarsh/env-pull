// Package executor handles sub-shell injection: it receives a resolved map of
// secret key-value pairs and spawns the caller's target command as a child
// process with those secrets present in the child's environment via OS-level
// process tree inheritance. Secrets are held only in memory and vanish when
// the child process terminates — they are never written to disk or logged.
package executor
