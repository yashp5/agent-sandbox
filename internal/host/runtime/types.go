package runtime

import "time"

// State describes a local runtime instance, not its public registry state.
type State uint8

const (
	// StateUnknown is the unset value and is not an operational state.
	StateUnknown State = iota
	// StateCreated has prepared resources but has not started execution.
	StateCreated
	// StateRunning has a running guest ready to accept commands.
	StateRunning
	// StatePaused has stopped execution while retaining running resources.
	// It is distinct from public hibernation, which releases those resources.
	StatePaused
)

// SnapshotMode is a resolved capture mode. Orchestration resolves the public
// AUTO mode before calling the driver and maps enum values explicitly.
type SnapshotMode uint8

const (
	// SnapshotModeUnknown is unset and is invalid for capture.
	SnapshotModeUnknown SnapshotMode = iota
	// SnapshotModeFull captures all guest memory.
	SnapshotModeFull
	// SnapshotModeDiff captures changed pages and requires a compatible base.
	SnapshotModeDiff
)

// ExecState describes a command session independently of its connections.
type ExecState uint8

const (
	// ExecStateUnknown is the unset value.
	ExecStateUnknown ExecState = iota
	// ExecStateRunning means the command is still executing.
	ExecStateRunning
	// ExecStateExited means the process ended; output may still be draining or
	// awaiting acknowledgement, and the final ExitStatus may not yet be available.
	ExecStateExited
)

// ProcessSignal selects a process-group signal. These are API values, not raw
// operating-system signal numbers; drivers map them explicitly.
type ProcessSignal uint8

const (
	// ProcessSignalUnknown is unset and cannot be sent.
	ProcessSignalUnknown ProcessSignal = iota
	// ProcessSignalInterrupt maps to SIGINT.
	ProcessSignalInterrupt
	// ProcessSignalTerminate maps to SIGTERM.
	ProcessSignalTerminate
	// ProcessSignalKill maps to SIGKILL.
	ProcessSignalKill
)

// ExecEventKind identifies the active payload in ExecEvent. Only the fields
// associated with that kind may be populated; the others must be zero or nil.
type ExecEventKind uint8

const (
	// ExecEventUnknown is unset and must not be emitted.
	ExecEventUnknown ExecEventKind = iota
	// ExecEventStdout carries stdout bytes in Offset and Data.
	ExecEventStdout
	// ExecEventStderr carries stderr bytes in Offset and Data.
	ExecEventStderr
	// ExecEventStdinAck carries the cumulative NextStdinSequence acknowledgement.
	ExecEventStdinAck
	// ExecEventExit carries a non-nil final ExitStatus after all output events.
	ExecEventExit
)

// SnapshotSpec identifies a new capture of an already-paused instance.
type SnapshotSpec struct {
	// ID is a nonempty, unused snapshot ID supplied by the caller.
	ID string
	// Mode must be SnapshotModeFull or SnapshotModeDiff.
	Mode SnapshotMode
}

// Resources is the resolved guest allocation. All values must be positive;
// template defaults are applied before calling the driver.
type Resources struct {
	// VCPUs is the number of virtual CPUs allocated to the guest.
	VCPUs uint32
	// MemoryMiB is configured guest RAM in mebibytes, not observed host usage.
	MemoryMiB uint32
	// DiskGiB is the provisioned guest disk capacity in gibibytes.
	DiskGiB uint32
}

// CreateSpec supplies resolved inputs for preparing an instance. A driver that
// retains the specification must copy its mutable data.
type CreateSpec struct {
	// ID is the nonempty, unused instance ID chosen by the caller.
	ID string
	// TemplateID selects the configured template and must be nonempty.
	TemplateID string
	// Resources contains concrete allocations rather than unspecified defaults.
	Resources Resources
	// Env supplies the sandbox environment inherited by exec commands.
	Env map[string]string
}

// Stats is an observation of instance state, configured allocation, and usage.
type Stats struct {
	// State is the instance state at observation time.
	State State
	// Resources is the configured guest allocation, not current consumption.
	Resources Resources
	// MemoryUsageBytes is host memory charged to the instance, including VMM
	// overhead. Shared snapshot memory charged elsewhere is not counted here.
	MemoryUsageBytes uint64
	// CPUTime is cumulative processor time consumed, not elapsed wall time.
	// Multiple busy CPUs can accumulate CPUTime faster than wall time passes.
	CPUTime time.Duration
}

// SnapshotInfo describes a retained capture. The driver stores the captured
// state, configuration, and any dependencies separately from this metadata.
type SnapshotInfo struct {
	// ID identifies the retained snapshot.
	ID string
	// SourceID identifies the instance whose state was captured.
	SourceID string
	// Mode reports the actual full or diff capture mode.
	Mode SnapshotMode
	// BaseSnapshotID identifies a required base, or is empty when none is needed.
	BaseSnapshotID string
	// CreatedAt records the capture timestamp.
	CreatedAt time.Time
}

// RestoreSpec selects retained state for a new paused instance. Restoration
// preserves the snapshot and uses its captured resource configuration.
type RestoreSpec struct {
	// ID is the nonempty, unused ID of the new instance.
	ID string
	// SnapshotID is the nonempty ID of the snapshot to restore.
	SnapshotID string
}

// ExecSpec starts a command whose lifetime is independent of the startup request
// context and RPC connections. A driver retaining it must copy mutable data.
type ExecSpec struct {
	// SessionID must be nonempty and unique for the entire instance lifetime.
	SessionID string
	// Argv contains the executable and arguments; its first element must be
	// nonempty. Shell interpretation requires explicitly invoking a shell.
	Argv []string
	// Env overrides the sandbox environment for this command.
	Env map[string]string
	// Cwd is a guest working directory; empty selects the template default.
	Cwd string
	// User is a guest username or numeric UID; empty selects the template default.
	User string
	// Timeout is the command execution limit. Zero adds no command-specific
	// deadline; negative values are invalid. Expiry terminates the process group.
	Timeout time.Duration
}

// ExitStatus is a final command result. Exactly one of ExitCode and SignalNumber
// must be non-nil; a non-nil ExitCode pointing to zero is a valid successful exit.
type ExitStatus struct {
	// ExitCode is present when the process exited normally.
	ExitCode *int32
	// SignalNumber is present when a signal terminated the process.
	SignalNumber *uint32
	// TimedOut reports termination initiated by the command timeout.
	TimedOut bool
	// StdoutEndOffset is the exclusive end offset: the total stdout byte count.
	StdoutEndOffset uint64
	// StderrEndOffset is the exclusive end offset: the total stderr byte count.
	StderrEndOffset uint64
}

// ExecSessionInfo is caller-owned session metadata. Reading it does not consume
// output or acknowledge completion. Mutable slices and pointed-to values must
// not alias the driver's live state.
type ExecSessionInfo struct {
	// ID is the session ID, scoped to its instance.
	ID string
	// State describes the process lifecycle rather than connection status.
	State ExecState
	// Argv is the command and its arguments.
	Argv []string
	// StartedAt records when execution started.
	StartedAt time.Time
	// Exit is nil until the final result and both output end offsets are known.
	Exit *ExitStatus
}

// AttachSpec selects a session and its replay starting positions. Drivers must
// copy supplied offset values rather than retaining caller-owned pointers.
type AttachSpec struct {
	// SessionID is nonempty and identifies a retained session in the instance.
	SessionID string
	// StdoutOffset requests the next stdout byte. Nil selects the oldest retained
	// byte; a pointer to zero explicitly requests the beginning. A discarded or
	// not-yet-produced position is ErrOutOfRange; the current end is valid.
	StdoutOffset *uint64
	// StderrOffset has the same semantics for the independent stderr stream.
	StderrOffset *uint64
}

// AttachmentInfo records the selected positions and guest input state when an
// attachment opens. It is an immutable observation, not a source of live counters.
type AttachmentInfo struct {
	// SessionID identifies the attached session within its instance.
	SessionID string
	// NextStdinSequence is the next expected sequence at opening, starting at 1.
	// Every smaller sequence has been accepted; this does not imply consumption.
	NextStdinSequence uint64
	// StdoutOffset is the actual next stdout byte selected for replay.
	StdoutOffset uint64
	// StderrOffset is the actual next stderr byte selected for replay.
	StderrOffset uint64
}

// StdinChunk is an ordered, replayable stdin submission. Sequences start at 1
// and advance by one. Retries reuse the original sequence, data, and EOF flag.
// Already accepted sequences are acknowledged without applying them again;
// sequences beyond the next expected value are rejected. Guest acceptance is
// into a bounded queue and is reported by ExecEventStdinAck.
type StdinChunk struct {
	// Sequence identifies this chunk, including a chunk that only closes stdin.
	Sequence uint64
	// Data contains at most 64 KiB (65,536 bytes). Only one new chunk may await
	// acknowledgement at a time; output acknowledgements and signals may continue.
	Data []byte
	// EOF closes process stdin after this chunk's bytes are written. It is
	// sequenced and deduplicated with the chunk; later new input is invalid.
	// Detaching a connection does not itself close stdin.
	EOF bool
}

// OutputAck identifies output the consumer has processed and no longer needs
// replayed. Both offsets are cumulative, exclusive, and nondecreasing: offset
// 100 acknowledges bytes 0 through 99. Neither may exceed delivered output.
type OutputAck struct {
	// StdoutOffset is the next stdout byte the consumer still needs.
	StdoutOffset uint64
	// StderrOffset is the next stderr byte the consumer still needs.
	StderrOffset uint64
	// ExitReceived acknowledges the final result. It is valid only once the
	// result is known and both offsets equal its final output byte counts.
	ExitReceived bool
}

// ExecEvent is a tagged union. Stdout/stderr use Offset and Data, a stdin
// acknowledgement uses NextStdinSequence, and exit uses a non-nil Exit. Other
// payload fields must be zero or nil. Data and nested ExitStatus values belong
// to the receiver and must not alias mutable driver state or reusable buffers.
type ExecEvent struct {
	// Kind selects the active payload and cannot be ExecEventUnknown.
	Kind ExecEventKind
	// Offset is the absolute first-byte position in the selected output stream;
	// stdout and stderr each start at zero and have independent positions.
	Offset uint64
	// Data contains at most 64 KiB (65,536 bytes) of binary output.
	Data []byte
	// NextStdinSequence acknowledges acceptance of all smaller stdin sequences,
	// not consumption by the process. Its value is at least 1.
	NextStdinSequence uint64
	// Exit is the final result, retained until explicitly acknowledged.
	Exit *ExitStatus
}
