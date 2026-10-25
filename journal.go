package saga

type Record struct {
	Node      nodeRunner
	Input     any
	Children  []*Record
	Result    any
	Succeeded bool
}

func (r *Record) HasNode() bool {
	return r != nil && r.Node != nil
}
