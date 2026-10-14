package saga

type Record[C any] struct {
	Node      *Node[C]
	Children  []*Record[C]
	Succeeded bool
}

func (r *Record[C]) HasNode() bool {
	if r == nil {
		return false
	}

	return r.Node != nil
}
