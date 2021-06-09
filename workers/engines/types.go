package engines

type Worker interface {
	Process(payload []byte) error
}
