package lib

// restartRequests carries a WebUI-initiated restart to the worker main loop.
// Buffered with room for one: extra requests while a restart is already
// pending are redundant, so they are dropped.
var restartRequests = make(chan string, 1)

// RequestRestart asks the worker to exit so the supervisor restarts it.
// It reports false when a restart is already pending.
func RequestRestart(reason string) bool {
	select {
	case restartRequests <- reason:
		return true
	default:
		return false
	}
}

// RestartRequests is the channel the worker main loop waits on.
func RestartRequests() <-chan string {
	return restartRequests
}
