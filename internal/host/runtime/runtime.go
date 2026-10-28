package runtime

import "context"

// ExecAttachment is a temporary connection to a persistent exec session. Closing
// or replacing it does not terminate the process, close its stdin, or acknowledge
// output. Each method's context bounds that operation, not the session lifetime.
//
// Callers use at most one Recv at a time. Controls and Close may run alongside
// Recv; Ack, Signal, and Close must remain usable while SendStdin is blocked.
// Implementations copy retained input buffers and return caller-owned event data.
type ExecAttachment interface {
	// Info returns the immutable opening metadata, not live counters. It remains
	// available after this attachment closes or is replaced.
	Info() AttachmentInfo
	// Recv waits for the next event, replaying output from the opening offsets.
	// Per-stream byte order is preserved; no ordering between stdout and stderr
	// is promised. Exit follows all output and supplies the final byte counts.
	//
	// Receiving Exit alone does not finish the attachment: Recv returns io.EOF
	// after the final exit acknowledgement is accepted. Before normal completion,
	// Close or replacement unblocks Recv with ErrAttachmentClosed. Cancellation
	// of this wait returns ctx.Err() without closing this attachment or terminating
	// its process.
	Recv(ctx context.Context) (ExecEvent, error)
	// SendStdin submits a sequenced chunk. A nil return reports submission;
	// ExecEventStdinAck confirms guest queue acceptance, not process consumption.
	// The client waits for that acknowledgement before sending the next new
	// chunk, but may still acknowledge output or signal the process while waiting.
	// Retained data is copied so the caller may reuse its buffer after return.
	// A failed or cancelled submission may already have been accepted; recovery
	// uses the original sequence and unchanged payload, not a new sequence.
	SendStdin(ctx context.Context, chunk StdinChunk) error
	// Signal sends a supported signal to the command's entire process group.
	// ProcessSignalUnknown is invalid. Signals are not sequenced or deduplicated
	// and must not be automatically replayed after an uncertain submission.
	Signal(ctx context.Context, signal ProcessSignal) error
	// Ack cumulatively acknowledges consumed output and, optionally, the final
	// result. Reject offsets that regress or exceed output delivered to the
	// consumer. Only acknowledged bytes may be discarded; full buffers apply
	// backpressure instead of dropping unacknowledged output.
	//
	// ExitReceived requires the known final result and both final output offsets.
	// Closed or superseded attachments cannot send stdin, signals, or acknowledgements;
	// those control methods return ErrAttachmentClosed.
	Ack(ctx context.Context, ack OutputAck) error
	// Close detaches this handle and unblocks its waiting Recv. It is idempotent
	// and does not affect a replacement attachment or the underlying process.
	// Unacknowledged output and the exit result remain available for reattachment.
	Close() error
}

// Runtime controls local instances and exec sessions independently of the public
// RPC transport. Callers supply stable instance, snapshot, and session IDs.
//
// Contexts bound operations and waiting; cancellation does not guarantee rollback
// of side effects that already occurred. Implementations return context errors
// when cancelled and preserve error identities for errors.Is. Retained inputs and
// returned metadata must not share mutable collections with callers.
type Runtime interface {
	// Create prepares resources and records spec.ID in StateCreated without
	// starting execution. IDs and template IDs must be nonempty, resources must
	// already be resolved to positive values, and the instance ID must be unused.
	// Invalid inputs return ErrInvalidConfig; an existing ID is ErrAlreadyExists.
	Create(ctx context.Context, spec CreateSpec) error
	// Start requires StateCreated and succeeds once the guest is ready for
	// commands in StateRunning. A missing ID is ErrNotFound; another state is
	// ErrInvalidState.
	Start(ctx context.Context, id string) error
	// Pause stops execution of a running instance while retaining its resources,
	// leaving it in StatePaused. A missing ID is ErrNotFound; another state is
	// ErrInvalidState.
	Pause(ctx context.Context, id string) error
	// Resume resumes a paused instance and returns it to StateRunning once the
	// guest is ready. A missing ID is ErrNotFound; another state is ErrInvalidState.
	Resume(ctx context.Context, id string) error
	// Destroy terminates execution, releases instance resources, and removes the
	// instance regardless of its state. An absent instance is already destroyed
	// and succeeds. Retained snapshots remain usable.
	Destroy(ctx context.Context, id string) error
	// Stats returns a caller-owned observation of state, allocation, and usage.
	// A missing instance is ErrNotFound.
	Stats(ctx context.Context, id string) (Stats, error)
	// Snapshot captures a paused instance under a nonempty, unused spec.ID and
	// leaves the source paused. The host owns pausing and resuming, including
	// arranging a resume attempt after capture errors. Diff captures require a
	// compatible base and tracking support. Captured state and required bases
	// remain usable after source destruction.
	//
	// A missing source is ErrNotFound; an existing snapshot ID is ErrAlreadyExists;
	// invalid inputs are ErrInvalidConfig; unsuitable state or a missing required
	// base is ErrInvalidState; a valid unsupported mode is ErrUnsupported.
	Snapshot(ctx context.Context, id string, spec SnapshotSpec) (SnapshotInfo, error)
	// Restore loads spec.SnapshotID into a new StatePaused instance using the
	// captured configuration. The snapshot remains available for later restores;
	// the new instance continues through Resume, not Create or Start.
	//
	// IDs must be nonempty. A missing snapshot is ErrNotFound; an existing target
	// instance is ErrAlreadyExists; invalid inputs are ErrInvalidConfig. The driver
	// must reject incompatible snapshots rather than restoring invalid state.
	Restore(ctx context.Context, spec RestoreSpec) error
	// Exec starts and registers a command in a running instance, then returns its
	// metadata without waiting for exit. A missing instance is ErrNotFound; another
	// state is ErrInvalidState; invalid input is ErrInvalidConfig. Reused session
	// IDs return ErrAlreadyExists, even after completed session data is discarded.
	//
	// ctx bounds startup waiting only. The process lifetime is independent of the
	// RPC connection and this context; spec.Timeout governs its execution limit.
	// A cancelled startup may already have started the session, recoverable by ID.
	Exec(ctx context.Context, id string, spec ExecSpec) (ExecSessionInfo, error)
	// ListExecSessions returns caller-owned metadata for running sessions and
	// completed sessions awaiting final acknowledgement. Reading metadata does
	// not acknowledge output or consume it. Used session IDs must remain recorded
	// until instance destruction, even after acknowledged session data is removed.
	// A missing instance is ErrNotFound.
	ListExecSessions(ctx context.Context, id string) ([]ExecSessionInfo, error)
	// Attach connects to a retained session in a running instance, including an
	// exited session whose final result is still unacknowledged. Missing instances
	// or sessions return ErrNotFound; a non-running instance is ErrInvalidState;
	// an empty session ID is ErrInvalidConfig. Unavailable offsets are ErrOutOfRange.
	//
	// Validate the request before replacing the active attachment. A successful
	// attachment supersedes the old handle, which must reject further controls
	// and unblock its receiver with ErrAttachmentClosed. Already accepted stdin
	// remains accepted, and closing the old handle cannot detach the new one.
	// ctx bounds opening only; subsequent calls use their own operation contexts.
	Attach(ctx context.Context, id string, spec AttachSpec) (ExecAttachment, error)
}
