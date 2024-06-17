package snapshot

import (
	"fmt"
	"sort"
	"time"
	"unsafe"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/DataExMachina-dev/side-eye-go/internal/allgs"
	"github.com/DataExMachina-dev/side-eye-go/internal/framing"
	"github.com/DataExMachina-dev/side-eye-go/internal/machinapb"
	"github.com/DataExMachina-dev/side-eye-go/internal/snapshotpb"
	"github.com/DataExMachina-dev/side-eye-go/internal/stackmachine"
	"github.com/DataExMachina-dev/side-eye-go/internal/stoptheworld"
)

type moduledataTypeRange struct {
	start uint64
	end   uint64
}

type moduledataConfig struct {
	runtimeDotFirstmoduledataAddr uintptr
	typesOffset                   uintptr
	etypesOffset                  uintptr
}

type goRuntimeTypeResolver struct {
	cfg                        moduledataConfig
	cachedFirstmoduledataRange moduledataTypeRange
	// TODO: Handle more than one moduledata. Go dynamically loaded libraries seem pretty rare.
}

func makeGoRuntimeTypeResolver(p *snapshotpb.RuntimeConfig) goRuntimeTypeResolver {
	return goRuntimeTypeResolver{
		cfg: moduledataConfig{
			runtimeDotFirstmoduledataAddr: uintptr(p.VariableRuntimeDotFirstmoduledata),
			typesOffset:                   uintptr(p.ModuledataTypesOffset),
			etypesOffset:                  uintptr(p.ModuledataEtypesOffset),
		},
	}
}

func (m *goRuntimeTypeResolver) maybeResolveFirstmoduledataRange() {
	if m.cachedFirstmoduledataRange.start != 0 {
		return
	}
	var v unsafe.Pointer
	moduledataPtr := unsafe.Pointer(uintptr(v) + m.cfg.runtimeDotFirstmoduledataAddr)
	types := *(*uint64)(unsafe.Pointer(uintptr(moduledataPtr) + m.cfg.typesOffset))
	etypes := *(*uint64)(unsafe.Pointer(uintptr(moduledataPtr) + m.cfg.etypesOffset))
	m.cachedFirstmoduledataRange = moduledataTypeRange{start: types, end: etypes}
}

func (m *goRuntimeTypeResolver) ResolveTypeAddressToGoRuntimeTypeId(addr uint64) uint64 {
	m.maybeResolveFirstmoduledataRange()
	r := m.cachedFirstmoduledataRange
	if addr < r.start || addr >= r.end {
		return 0
	}
	return addr - r.start
}

func Snapshot(p *snapshotpb.SnapshotProgram) (*machinapb.SnapshotResponse, error) {
	b := newSnapshotter(p)
	start := time.Now()
	snapshotHeader, ok := b.out.writeSnapshotHeader()
	if !ok {
		return nil, fmt.Errorf("failed to write snapshot header")
	}
	stoptheworld.StopTheWorld(p.RuntimeConfig, func() {
		allgs.ForEach(p.RuntimeConfig, func(g allgs.Goroutine) {
			b.snapshotGoroutine(snapshotHeader, g)
		})

		afterStacks := time.Now()
		snapshotHeader.Statistics.StacksDurationNs = uint64(afterStacks.Sub(start).Nanoseconds())
		snapshotHeader.GoroutinesByteLen = b.out.Len() - uint32(unsafe.Sizeof(framing.SnapshotHeader{}))

		b.processQueue()
		snapshotHeader.Statistics.PointerDurationNs = uint64(time.Since(afterStacks).Nanoseconds())
	})
	snapshotHeader.DataByteLen = b.out.Len()
	snapshotHeader.Statistics.TotalDurationNs = uint64(time.Since(start).Nanoseconds())
	return &machinapb.SnapshotResponse{
		Data:            b.out.data(),
		Timestamp:       timestamppb.New(start),
		PauseDurationNs: snapshotHeader.Statistics.TotalDurationNs,
	}, nil
}

func newSnapshotter(p *snapshotpb.SnapshotProgram) *snapshotter {
	var b snapshotter
	b.p = p
	b.stacks = make(map[uint64][]frameOfInterest, 512 /* arbitrary */)
	b.out = makeOutBuf(1 << 20)
	b.queue = makeQueue()
	b.unwinder = newUnwinder()
	b.goRuntimeTypeResolver = makeGoRuntimeTypeResolver(p.RuntimeConfig)
	b.typeIdResolver = typeIdResolver{types: p.GoRuntimeTypeToTypeId}
	b.sm = stackmachine.New(p.Prog, &b.queue, &b.out, &b.goRuntimeTypeResolver, &b.typeIdResolver)
	return &b
}

const maxStackFrames = 512

type snapshotter struct {
	stacks                map[uint64][]frameOfInterest
	goRuntimeTypeResolver goRuntimeTypeResolver
	typeIdResolver        typeIdResolver
	out                   outBuf
	queue                 queue
	unwinder              *unwinder
	p                     *snapshotpb.SnapshotProgram
	sm                    *stackmachine.StackMachine[*queue, *outBuf, *goRuntimeTypeResolver, *typeIdResolver]
}

func (s *snapshotter) snapshotGoroutine(snapshotHeader *framing.SnapshotHeader, g allgs.Goroutine) {
	if s.out.full() {
		return
	}
	defer func(before uint32) {
		if s.out.full() {
			s.out.truncate(before)
		}
	}(s.out.Len())

	status := g.Status() & (^allgs.Status(allgs.Status_Gscan))
	if status == allgs.Status_Gdead {
		snapshotHeader.Statistics.NonLiveGoroutines++
		return
	}
	// This is our goroutine, we can't unwind it because we don't have the context.
	// Also, we don't care to.
	if status == allgs.Status_Grunning {
		return
	}

	snapshotHeader.Statistics.NumGoroutines++

	// TODO(https://github.com/DataExMachina-dev/side-eye/issues/756): This
	// should use syscallpc and the syscall frame, but in go versions before
	// 1.23, the syscall base pointer is not recorded. Some degree of unwinding
	// is needed.
	pcs, fps := s.unwinder.walkStack(g.PC(), g.BP())
	stackHash := murmur2(pcs, 0)
	goroutineHeader, ok := s.out.writeGoroutineHeader()
	if !ok {
		return
	}
	afterHeader := s.out.Len()
	framesOfInterest, haveStack := s.stacks[stackHash]
	var stackBytes uint32

	// If the stack with this hash isn't in the output, write it, and
	// classify the frames of interest.
	if !haveStack {
		stackBytes, ok = s.out.writeStack(pcs)
		if !ok {
			return
		}
		for i := range pcs {
			pc := uint64(pcs[i])
			j := sort.Search(len(s.p.PcClassifier.TargetPc), func(j int) bool {
				return pc <= s.p.PcClassifier.TargetPc[j]
			})
			if j < len(s.p.PcClassifier.TargetPc) && s.p.PcClassifier.ProgPc[j] != 0 {
				framesOfInterest = append(framesOfInterest, frameOfInterest{
					idx: uint32(i),
					pc:  s.p.PcClassifier.ProgPc[j],
				})
			}
		}
		s.stacks[stackHash] = framesOfInterest
	}

	// Run the stack machine program to write out the data from the stack
	// frames and enqueue the pointers.
	for _, foi := range framesOfInterest {
		if !s.sm.Run(foi.pc, fps[foi.idx], foi.idx, s.out.Len()) {
			break
		}
	}

	*goroutineHeader = framing.GoroutineHeader{
		Goid:           g.Goid(),
		StackHash:      stackHash,
		Status:         uint32(status),
		WaitReason:     0,
		WaitSinceNanos: 0,
		StackBytes:     stackBytes,
		DataByteLen:    s.out.Len() - afterHeader,
	}
}

func (s *snapshotter) processQueue() {
	for !s.out.full() {
		entry, ok := s.queue.Pop()
		if !ok {
			break
		}
		ti, ok := s.p.TypeInfo[entry.Type]
		if !ok {
			continue
		}
		if entry.Len == 0 {
			entry.Len = ti.ByteLen
		}
		if entry.Len > ti.ByteLen {
			entry.Len = ti.ByteLen
		}
		if entry.Len == 0 {
			continue
		}
		offset, ok := s.out.writeQueueEntry(entry)
		if !ok {
			continue
		}
		if ti.EnqueuePc == 0 {
			continue
		}
		s.sm.Run(ti.EnqueuePc, 0, 0, offset)
	}
}

type typeIdResolver struct {
	types map[uint64]uint32
}

func (r *typeIdResolver) ResolveTypeAddressToTypeId(addr uint64) uint32 {
	return r.types[addr]
}

// Below code taken and lightly modified from
// https://github.com/parca-dev/parca-agent/blob/aa9289b868/bpf/unwinders/hash.h
//
// murmurhash2 from
// https://github.com/aappleby/smhasher/blob/92cf3702fcfaadc84eb7bef59825a23e0cd84f56/src/MurmurHash2.cpp
func murmur2(stack []uintptr, seed uint64) uint64 {
	const m = uint64(0xc6a4a7935bd1e995)
	const r = 47
	hash := seed ^ (uint64(len(stack)) * m)
	for i := 0; i < len(stack); i++ {
		k := uint64(stack[i])
		k *= m
		k ^= k >> r
		k *= m

		hash ^= k
		hash *= m
	}
	return hash
}

type frameOfInterest struct {
	idx uint32
	pc  uint32
}
