package ns

// Windows file handles are flushed by the engine data path. There is no
// process-wide equivalent to Linux sync(2).
func syncFilesystem() error { return nil }
