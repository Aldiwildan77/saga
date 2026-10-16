package saga

type Record[I, O any] struct {
	Node      *Node[I, O]
	Children  []*Record[I, O]
	Result    O
	Succeeded bool
}

func (r *Record[I, O]) HasNode() bool {
	if r == nil {
		return false
	}

	return r.Node != nil
}
