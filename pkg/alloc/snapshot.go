package alloc

import "fmt"

// WriteChange records a single contiguous byte region that changed between
// two consecutive snapshots. Old and New are both len(New) == New-Old bytes.
type WriteChange struct {
	Offset int
	Old    []byte
	New    []byte
}

// SnapshotDelta is the incremental diff between two consecutive allocator states.
// It is returned by Take() and can be used directly by the TUI for rendering.
//
//   - Writes: byte regions whose content changed (store/overwrite operations)
//   - Consumed: free blocks that disappeared (memory was allocated)
//   - Freed: free blocks that appeared (memory was returned to the pool)
type SnapshotDelta struct {
	HistoryID int
	Writes    []WriteChange
	Consumed  []Region
	Freed     []Region
}

// IsEmpty reports whether the delta contains no changes.
func (d SnapshotDelta) IsEmpty() bool {
	return len(d.Writes) == 0 && len(d.Consumed) == 0 && len(d.Freed) == 0
}

// SnapshotHistory is a single point in the snapshot history.
// It stores the full free list at that moment and the buffer delta
// against the previous entry, keeping memory cost proportional to
// the number of bytes that actually changed.
type SnapshotHistory struct {
	ID           int
	BufferSize   int           // actual buffer size at this snapshot
	FreedRegions []Region      // full free list at this point in time
	Writes       []WriteChange // buffer regions that changed since the previous entry
	Consumed     []Region      // free blocks consumed since the previous entry
	Freed        []Region      // free blocks freed since the previous entry
}

// SnapshotManager maintains an ordered, incremental snapshot history for a
// single allocator. The baseline is captured on construction; every subsequent
// Take() stores only the delta against the preceding snapshot.
//
// Memory cost: one full buffer copy for the base, plus the sum of all changed
// byte regions across subsequent snapshots — not N full copies.
type SnapshotManager struct {
	target    Snapshottable
	initial   Snapshot // initial full snapshot — never mutated after construction
	prev      Snapshot // deep copy of the most recent snapshot, used for diffing
	histories []SnapshotHistory
	nextID    int
}

// NewSnapshotManager creates a SnapshotManager and captures the allocator's
// current state as the baseline. All subsequent Take() calls produce deltas
// relative to the preceding snapshot.
func NewSnapshotManager(target Snapshottable) (*SnapshotManager, error) {
	initial, err := target.Snapshot()
	if err != nil {
		return nil, fmt.Errorf("failed to create initial snapshot: %v", err)
	}
	// Deep-copy base into prev so that mutations to base.Buffer via Restore
	// never corrupt the diff baseline.
	prevBuf := make([]byte, len(initial.Buffer))
	copy(prevBuf, initial.Buffer)
	prevFree := make([]Region, len(initial.FreedRegions))
	copy(prevFree, initial.FreedRegions)

	return &SnapshotManager{
		target:  target,
		initial: initial,
		prev: Snapshot{
			Buffer:       prevBuf,
			FreedRegions: prevFree,
			Capacity:     initial.Capacity,
		},
	}, nil
}

// Take captures the allocator's current state, computes the delta against the
// previous snapshot, appends it to the history, and returns the new entry's
// ID and its delta. The first call after construction uses the baseline as the
// predecessor, so it reports everything that changed since session start.
func (m *SnapshotManager) Take() (SnapshotDelta, error) {
	cur, err := m.target.Snapshot()
	if err != nil {
		return SnapshotDelta{}, fmt.Errorf("failed to capture snapshot state: %v", err)
	}

	writes := diffBuffers(m.prev.Buffer, cur.Buffer)
	consumed, freed := diffFreedRegionss(m.prev.FreedRegions, cur.FreedRegions)

	freeCopy := make([]Region, len(cur.FreedRegions))
	copy(freeCopy, cur.FreedRegions)

	history := SnapshotHistory{
		ID:           m.nextID,
		BufferSize:   len(cur.Buffer),
		FreedRegions: freeCopy,
		Writes:       writes,
		Consumed:     consumed,
		Freed:        freed,
	}
	m.histories = append(m.histories, history)
	m.nextID++

	// Advance prev to current for the next diff.
	m.prev = cur

	return SnapshotDelta{
		HistoryID: history.ID,
		Writes:    writes,
		Consumed:  consumed,
		Freed:     freed,
	}, nil
}

// Restore reconstructs the allocator state at the given snapshot ID by
// replaying the base buffer with all deltas up to and including that ID,
// then writing the result back to the target allocator.
//
// Entries after id are pruned — they are no longer consistent with the
// restored state.
func (m *SnapshotManager) Restore(historyId int) error {
	if historyId < 0 || historyId >= len(m.histories) {
		return fmt.Errorf("failed to restore snapshot - id %d for history is out of range [0, %d)", historyId, len(m.histories))
	}

	// Reconstruct the buffer at the size it had at historyId, then apply deltas.
	bufSize := m.histories[historyId].BufferSize
	if bufSize == 0 {
		bufSize = len(m.initial.Buffer)
	}
	buf := make([]byte, bufSize)
	copy(buf, m.initial.Buffer)
	for _, entry := range m.histories[:historyId+1] {
		for _, w := range entry.Writes {
			copy(buf[w.Offset:], w.New)
		}
	}

	snap := Snapshot{
		Buffer:       buf,
		FreedRegions: m.histories[historyId].FreedRegions,
		Capacity:     m.initial.Capacity,
	}
	m.target.Restore(snap)

	// Reset prev to the reconstructed state so the next Take() diffs correctly.
	prevBuf := make([]byte, len(buf))
	copy(prevBuf, buf)
	prevFree := make([]Region, len(snap.FreedRegions))
	copy(prevFree, snap.FreedRegions)
	m.prev = Snapshot{
		Buffer:       prevBuf,
		FreedRegions: prevFree,
		Capacity:     m.initial.Capacity,
	}

	// Prune history: entries after id are now stale.
	m.histories = m.histories[:historyId+1]
	m.nextID = historyId + 1

	return nil
}

// History returns all snapshot entries in chronological order.
func (m *SnapshotManager) History() []SnapshotHistory {
	return m.histories
}

// Latest returns the most recent snapshot entry, or nil if Take has never
// been called.
func (m *SnapshotManager) Latest() *SnapshotHistory {
	if len(m.histories) == 0 {
		return nil
	}
	return &m.histories[len(m.histories)-1]
}

// Len returns the number of snapshots taken so far.
func (m *SnapshotManager) Len() int {
	return len(m.histories)
}

// Current returns a deep copy of the allocator's most recently observed state.
// Before the first Take() this reflects the baseline captured at construction.
// The returned AllocSnapshot is safe to read concurrently with future Take() calls.
func (m *SnapshotManager) Current() Snapshot {
	buf := make([]byte, len(m.prev.Buffer))
	copy(buf, m.prev.Buffer)
	free := make([]Region, len(m.prev.FreedRegions))
	copy(free, m.prev.FreedRegions)
	return Snapshot{
		Buffer:       buf,
		FreedRegions: free,
		Capacity:     m.prev.Capacity,
	}
}

// diffBuffers computes the changed byte regions between two buffer snapshots.
// Adjacent changed bytes are merged into a single WriteChange.
// If curr is longer than prev (buffer grew), any non-zero bytes in the new
// region are also captured so they can be replayed during Restore.
func diffBuffers(prev, curr []byte) []WriteChange {
	var changes []WriteChange
	n := min(len(prev), len(curr))

	i := 0
	for i < n {
		if prev[i] == curr[i] {
			i++
			continue
		}
		start := i
		for i < n && prev[i] != curr[i] {
			i++
		}
		old := make([]byte, i-start)
		copy(old, prev[start:i])
		newB := make([]byte, i-start)
		copy(newB, curr[start:i])
		changes = append(changes, WriteChange{
			Offset: start,
			Old:    old,
			New:    newB,
		})
	}

	// Capture non-zero bytes in the growth region (curr grew beyond prev).
	if len(curr) > len(prev) {
		added := curr[len(prev):]
		j := 0
		for j < len(added) {
			if added[j] == 0 {
				j++
				continue
			}
			start := j
			for j < len(added) && added[j] != 0 {
				j++
			}
			old := make([]byte, j-start)
			newB := make([]byte, j-start)
			copy(newB, added[start:j])
			changes = append(changes, WriteChange{
				Offset: len(prev) + start,
				Old:    old,
				New:    newB,
			})
		}
	}

	return changes
}

// diffFreedRegionss returns the set difference between two free lists.
// consumed = blocks in prev that are absent in curr (were allocated).
// freed    = blocks in curr that are absent in prev (were returned).
func diffFreedRegionss(prev, curr []Region) (consumed, freed []Region) {
	prevSet := make(map[Region]struct{}, len(prev))
	for _, b := range prev {
		prevSet[b] = struct{}{}
	}
	currSet := make(map[Region]struct{}, len(curr))
	for _, b := range curr {
		currSet[b] = struct{}{}
	}

	for _, b := range prev {
		if _, ok := currSet[b]; !ok {
			consumed = append(consumed, b)
		}
	}
	for _, b := range curr {
		if _, ok := prevSet[b]; !ok {
			freed = append(freed, b)
		}
	}
	return
}
