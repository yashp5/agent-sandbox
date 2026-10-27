package runtime

type Runtime interface {
	Create()
	Start()
	Exec()
	Pause()
	Resume()
	Snapshot()
	Restore()
	Destroy()
	Stats()
}
