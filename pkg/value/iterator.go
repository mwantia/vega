package value

// --- iterator ---

type sliceIterator struct {
	data []byte
	pos  int
}

func (it *sliceIterator) Next() bool {
	return it.pos < len(it.data)
}

func (it *sliceIterator) Value() Value {
	b := it.data[it.pos]
	it.pos++

	return NewByte([]byte{b})
}
