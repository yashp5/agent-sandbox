package runtime

import "errors"

var (
	// ErrNotFound identifies a requested instance, snapshot, or session that is
	// absent. Callers should use errors.Is, including when drivers wrap the error.
	ErrNotFound = errors.New("runtime instance not found")
	// ErrAlreadyExists identifies an ID that is already in use. Used exec session
	// IDs remain reserved until their instance is destroyed.
	ErrAlreadyExists = errors.New("runtime instance already exists")
	// ErrInvalidState identifies an unmet runtime precondition, such as pausing a
	// non-running instance or requesting a diff without the required base.
	ErrInvalidState = errors.New("invalid runtime state")
	// ErrInvalidConfig identifies invalid inputs regardless of current state.
	ErrInvalidConfig = errors.New("invalid runtime configuration")
	// ErrUnsupported identifies a valid operation or mode that the driver does
	// not implement, rather than an invalid input or a temporary state conflict.
	ErrUnsupported = errors.New("runtime operation not supported")
	// ErrOutOfRange identifies a requested output position outside the retained
	// replay interval, including discarded bytes and positions beyond its end.
	// Failed attachment validation must not replace an existing attachment.
	ErrOutOfRange = errors.New("exec output offset out of range")
	// ErrAttachmentClosed identifies a locally closed or superseded handle.
	// It does not imply that the process exited or its session was destroyed.
	// Normal stream completion after the final acknowledgement uses io.EOF.
	ErrAttachmentClosed = errors.New("exec attachment closed or replaced")
)
