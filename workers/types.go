package workers

type Worker interface {
	Process(payload []byte)
}
