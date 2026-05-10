package engine

import (
	"fmt"
	"strings"
	"sync/atomic"
	"unsafe"
)

type TTEntry struct {
	key    atomic.Uint64
	key2   atomic.Uint64
	packed atomic.Uint64
}

type ttEntryData struct {
	Key   Key
	Key2  Key
	Score int16
	Depth uint8
	Flag  int8
	Move  MoveNG
	Age   uint8
}

func (entry *TTEntry) Load() (ttEntryData, bool) {
	key := Key(entry.key.Load())
	if key == 0 {
		return ttEntryData{}, false
	}

	key2 := Key(entry.key2.Load())
	packed := entry.packed.Load()
	return unpackTTEntry(key, key2, packed), true
}

func (entry *TTEntry) Store(data ttEntryData) {
	entry.key2.Store(uint64(data.Key2))
	entry.packed.Store(packTTEntry(data))
	entry.key.Store(uint64(data.Key))
}

func (entry *TTEntry) Clear() {
	entry.key.Store(0)
	entry.key2.Store(0)
	entry.packed.Store(0)
}

func packTTEntry(data ttEntryData) uint64 {
	return uint64(uint16(data.Score)) |
		uint64(data.Depth)<<16 |
		uint64(uint8(data.Flag))<<24 |
		uint64(data.Age)<<32 |
		uint64(uint16(data.Move))<<40
}

func unpackTTEntry(key, key2 Key, packed uint64) ttEntryData {
	return ttEntryData{
		Key:   key,
		Key2:  key2,
		Score: int16(uint16(packed)),
		Depth: uint8(packed >> 16),
		Flag:  int8(uint8(packed >> 24)),
		Age:   uint8(packed >> 32),
		Move:  MoveNG(uint16(packed >> 40)),
	}
}

const (
	TT_ALPHA int8 = 1 + iota
	TT_BETA
	TT_EXACT
)

type TransTable struct {
	Entries []TTEntry
	Mask    uint64
}

var (
	TT  = NewTranTable(16)
	age atomic.Uint32
)

func roundPowerOfTwo(size int) int {
	x := 1
	for (x << 1) <= size {
		x <<= 1
	}
	return x
}

func NewTranTable(megabytes int) *TransTable {
	megabytes = min(4096, max(1, megabytes))
	size := 1024 * 1024 * megabytes / int(unsafe.Sizeof(TTEntry{}))
	tableSize := roundPowerOfTwo(size)
	return &TransTable{
		Entries: make([]TTEntry, tableSize),
		Mask:    uint64(tableSize - 1),
	}
}

func TTClear() {
	age.Store(0)
	for i := range TT.Entries {
		TT.Entries[i].Clear()
	}
}

func UpdateAge() {
	age.Add(1)
}

func TTSave(key, key2 Key, score int16, flag int8, ply, depth uint8, move MoveNG) {
	entry := &TT.Entries[key&TT.Mask]
	stored, ok := entry.Load()
	currentAge := uint8(age.Load())
	if shouldReplaceTTEntry(stored, ok, depth, flag, currentAge) {
		// If the score we get from the transposition table is a checkmate score, we need
		// to do a little extra work. This is because we store checkmates in the table using
		// their distance from the node they're found in, not their distance from the root.
		// So if we found a checkmate-in-8 in a node that was 5 plies from the root, we need
		// to store the score as a checkmate-in-3. Then, if we read the checkmate-in-3 from
		// the table in a node that's 4 plies from the root, we need to return the score as
		// checkmate-in-7.
		if score > int16(VALUE_MATE_IN_MAX_PLY) {
			score += int16(ply)
		} else if score < -int16(VALUE_MATE_IN_MAX_PLY) {
			score -= int16(ply)
		}
		entry.Store(ttEntryData{
			Key:   key,
			Key2:  key2,
			Score: score,
			Depth: depth,
			Flag:  flag,
			Move:  move,
			Age:   currentAge,
		})
	}
}

// The TT uses one lock-free atomic slot per index and a depth-preferred
// replacement policy. Deeper entries are kept unless the new entry is deep
// enough, the stored entry is stale, or the new bound is exact.
func shouldReplaceTTEntry(stored ttEntryData, ok bool, depth uint8, flag int8, currentAge uint8) bool {
	if !ok {
		return true
	}
	if stored.Age != currentAge && flag == TT_EXACT {
		return true
	}

	ageDelta := int(currentAge) - int(stored.Age)
	if ageDelta < 0 {
		ageDelta += 256
	}
	storedPriority := int(stored.Depth) - 2*ageDelta
	if flag != TT_EXACT && stored.Flag == TT_EXACT {
		storedPriority++
	}
	return storedPriority <= int(depth)
}

func TTProbe(key, key2 Key) (ttEntryData, bool) {
	entry := &TT.Entries[key&TT.Mask]
	data, ok := entry.Load()
	if ok && data.Key == key && data.Key2 == key2 {
		return data, true
	}
	return ttEntryData{}, false
}

func PrintTT(depth uint8) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Transposition Table Contents (depth %d):\n", depth))
	for index := range TT.Entries {
		entry := &TT.Entries[index]
		data, ok := entry.Load()
		if !ok || data.Key == 0 {
			continue
		}
		result.WriteString(fmt.Sprintf(
			"tt[%d]: key=%d key2=%d depth=%d flag=%d score=%d move=%d age=%d\n",
			index,
			data.Key,
			data.Key2,
			data.Depth,
			data.Flag,
			data.Score,
			data.Move,
			data.Age,
		))
	}
	return result.String()
}
