package snapshot

import (
	"unsafe"

	"github.com/DataExMachina-dev/side-eye-go/internal/stoptheworld"
)

type unwinder struct {
	numFrames int
	pcBuf     [maxStackFrames]uintptr
	fpBuf     [maxStackFrames]uintptr

	// Used during unwinding to avoid allocations.
	frameBuf      callFrame
	frameBufSlice []byte
}

type callFrame struct {
	fp uintptr
	pc uintptr
}

func newUnwinder() *unwinder {
	uw := new(unwinder)
	uw.frameBufSlice = unsafe.Slice(
		(*byte)(unsafe.Pointer(&uw.frameBuf)),
		unsafe.Sizeof(callFrame{}),
	)
	return uw
}

func (b *unwinder) walkStack(pc uintptr, fp uintptr) (pcs []uintptr, fps []uintptr) {
	b.pcBuf[0] = pc
	b.fpBuf[0] = fp
	b.numFrames = 1
	if fp == 0 {
		return b.pcBuf[:b.numFrames], b.fpBuf[:b.numFrames]
	}
	for ; b.numFrames < maxStackFrames; b.numFrames++ {
		nextCallFrame := b.fpBuf[b.numFrames-1]
		if !stoptheworld.Dereference(
			b.frameBufSlice,
			nextCallFrame,
			int(unsafe.Sizeof(callFrame{})),
		) {
			break
		}

		b.fpBuf[b.numFrames] = b.frameBuf.fp
		b.pcBuf[b.numFrames] = b.frameBuf.pc
		if b.fpBuf[b.numFrames] == 0 {
			break
		}
	}
	return b.pcBuf[:b.numFrames], b.fpBuf[:b.numFrames]
}