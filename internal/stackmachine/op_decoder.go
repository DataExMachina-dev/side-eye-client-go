package stackmachine

import (
	"encoding/binary"
	"fmt"
)

// OpDecoder is a decoder for stack machine operations.
type OpDecoder struct {
	pc    uint32
	opBuf []byte
}

// MakeOpDecoder creates a new OpDecoder.
func MakeOpDecoder(opBuf []byte) OpDecoder {
	return OpDecoder{
		pc:    0,
		opBuf: opBuf,
	}
}

// SetPC sets the program counter of the decoder.
func (d *OpDecoder) SetPC(pc uint32) bool {
	if pc >= uint32(len(d.opBuf)) {
		return false
	}
	d.pc = pc
	return true
}

// PC returns the program counter of the decoder.
func (d *OpDecoder) PC() uint32 {
	return d.pc
}

type OpCode uint8

//go:generate go run golang.org/x/tools/cmd/stringer@v0.22.0 -type=OpCode
const (
	OpCodeInvalid               OpCode = 0
	OpCodeCall                  OpCode = 1
	OpCodeCondJump              OpCode = 2
	OpCodeDecrement             OpCode = 3
	OpCodeEnqueueEmptyInterface OpCode = 4
	OpCodeEnqueueInterface      OpCode = 5
	OpCodeEnqueuePointer        OpCode = 6
	OpCodeEnqueueSliceHeader    OpCode = 7
	OpCodeEnqueueStringHeader   OpCode = 8
	OpCodeEnqueueMapHeader      OpCode = 9
	OpCodeJump                  OpCode = 10
	OpCodePop                   OpCode = 11
	OpCodePushImm               OpCode = 12
	OpCodePushOffset            OpCode = 13
	OpCodePushSliceLen          OpCode = 14
	OpCodeReturn                OpCode = 15
	OpCodeSetOffset             OpCode = 16
	OpCodeShiftOffset           OpCode = 17
	OpCodeEnqueueBiasedPointer  OpCode = 18
	OpCodeDereferenceCFAOffset  OpCode = 19
	OpCodeCopyFromRegister      OpCode = 20
	OpCodeZeroFill              OpCode = 21
	OpCodePrepareFrameData      OpCode = 22
)

type (
	OpCall struct {
		Pc uint32
	}
	OpCondJump struct {
		Pc uint32
	}
	OpDecrement             struct{}
	OpEnqueueEmptyInterface struct{}
	OpEnqueueInterface      struct{}
	OpEnqueuePointer        struct {
		ElemType uint32
	}
	OpEnqueueSliceHeader struct {
		ArrayType   uint32
		ElemByteLen uint32
	}
	OpEnqueueStringHeader struct {
		StringDataType uint32
	}
	OpEnqueueMapHeader struct {
		BucketsArrayType uint32
		BucketByteLen    uint32
		FlagsOffset      uint8
		BOffset          uint8
		BucketsOffset    uint8
		OldBucketsOffset uint8
	}
	OpJump struct {
		Pc uint32
	}
	OpPop     struct{}
	OpPushImm struct {
		Value uint32
	}
	OpPushOffset   struct{}
	OpPushSliceLen struct {
		ElemByteLen uint32
	}
	OpReturn      struct{}
	OpSetOffset   struct{}
	OpShiftOffset struct {
		Increment uint32
	}
	OpEnqueueBiasedPointer struct {
		ElemType uint32
		Bias     uint32
	}
	OpDereferenceCFAOffset struct {
		Offset      int32
		ByteLen     uint32
		PointerBias uint32
	}
	OpCopyFromRegister struct {
		Register uint16
	}
	OpZeroFill struct {
		ByteLen uint32
	}
	OpPrepareFrameData struct {
		ProgID      uint32
		DataByteLen uint32
		TypeID      uint32
	}
)

func (d *OpDecoder) PopOpCode() OpCode {
	code := OpCode(d.opBuf[d.pc])
	d.pc += 1
	return code
}

func (d *OpDecoder) DecodeCall() OpCall {
	pc := binary.LittleEndian.Uint32(d.opBuf[d.pc:])
	d.pc += 4
	return OpCall{
		Pc: pc,
	}
}

func (d *OpDecoder) DecodeCondJump() OpCondJump {
	pc := binary.LittleEndian.Uint32(d.opBuf[d.pc:])
	d.pc += 4
	return OpCondJump{
		Pc: pc,
	}
}
func (d *OpDecoder) DecodeDecrement() OpDecrement {
	return OpDecrement{}
}
func (d *OpDecoder) DecodeEnqueueEmptyInterface() OpEnqueueEmptyInterface {
	return OpEnqueueEmptyInterface{}
}
func (d *OpDecoder) DecodeEnqueueInterface() OpEnqueueInterface {
	return OpEnqueueInterface{}
}
func (d *OpDecoder) DecodeEnqueuePointer() OpEnqueuePointer {
	elemType := binary.LittleEndian.Uint32(d.opBuf[d.pc:])
	d.pc += 4
	return OpEnqueuePointer{
		ElemType: elemType,
	}
}
func (d *OpDecoder) DecodeEnqueueSliceHeader() OpEnqueueSliceHeader {
	arrayType := binary.LittleEndian.Uint32(d.opBuf[d.pc:])
	elemByteLen := binary.LittleEndian.Uint32(d.opBuf[d.pc+4:])
	d.pc += 8
	return OpEnqueueSliceHeader{
		ArrayType:   arrayType,
		ElemByteLen: elemByteLen,
	}
}
func (d *OpDecoder) DecodeEnqueueStringHeader() OpEnqueueStringHeader {
	stringDataType := binary.LittleEndian.Uint32(d.opBuf[d.pc:])
	d.pc += 4
	return OpEnqueueStringHeader{
		StringDataType: stringDataType,
	}
}
func (d *OpDecoder) DecodeEnqueueMapHeader() OpEnqueueMapHeader {
	op := OpEnqueueMapHeader{
		BucketsArrayType: binary.LittleEndian.Uint32(d.opBuf[d.pc:]),
		BucketByteLen:    binary.LittleEndian.Uint32(d.opBuf[d.pc+4:]),
		FlagsOffset:      d.opBuf[d.pc+8],
		BOffset:          d.opBuf[d.pc+9],
		BucketsOffset:    d.opBuf[d.pc+10],
		OldBucketsOffset: d.opBuf[d.pc+11],
	}
	d.pc += 12
	return op
}
func (d *OpDecoder) DecodeJump() OpJump {
	pc := binary.LittleEndian.Uint32(d.opBuf[d.pc:])
	d.pc += 4
	return OpJump{
		Pc: pc,
	}
}
func (d *OpDecoder) DecodePop() OpPop {
	return OpPop{}
}
func (d *OpDecoder) DecodePushImm() OpPushImm {
	value := binary.LittleEndian.Uint32(d.opBuf[d.pc:])
	d.pc += 4
	return OpPushImm{
		Value: value,
	}
}
func (d *OpDecoder) DecodePushOffset() OpPushOffset {
	return OpPushOffset{}
}
func (d *OpDecoder) DecodePushSliceLen() OpPushSliceLen {
	elemByteLen := binary.LittleEndian.Uint32(d.opBuf[d.pc:])
	d.pc += 4
	return OpPushSliceLen{
		ElemByteLen: elemByteLen,
	}
}
func (d *OpDecoder) DecodeReturn() OpReturn {
	return OpReturn{}
}
func (d *OpDecoder) DecodeSetOffset() OpSetOffset {
	return OpSetOffset{}
}
func (d *OpDecoder) DecodeShiftOffset() OpShiftOffset {
	increment := binary.LittleEndian.Uint32(d.opBuf[d.pc:])
	d.pc += 4
	return OpShiftOffset{
		Increment: increment,
	}
}
func (d *OpDecoder) DecodeEnqueueBiasedPointer() OpEnqueueBiasedPointer {
	elemType := binary.LittleEndian.Uint32(d.opBuf[d.pc:])
	bias := binary.LittleEndian.Uint32(d.opBuf[d.pc+4:])
	d.pc += 8
	return OpEnqueueBiasedPointer{
		ElemType: elemType,
		Bias:     bias,
	}
}
func (d *OpDecoder) DecodeDereferenceCFAOffset() OpDereferenceCFAOffset {
	offset := int32(binary.LittleEndian.Uint32(d.opBuf[d.pc:]))
	byteLen := binary.LittleEndian.Uint32(d.opBuf[d.pc+4:])
	pointerBias := binary.LittleEndian.Uint32(d.opBuf[d.pc+8:])
	d.pc += 12
	return OpDereferenceCFAOffset{
		Offset:      offset,
		ByteLen:     byteLen,
		PointerBias: pointerBias,
	}
}
func (d *OpDecoder) DecodeCopyFromRegister() OpCopyFromRegister {
	register := binary.LittleEndian.Uint16(d.opBuf[d.pc:])
	d.pc += 2
	return OpCopyFromRegister{
		Register: register,
	}
}
func (d *OpDecoder) DecodeZeroFill() OpZeroFill {
	byteLen := binary.LittleEndian.Uint32(d.opBuf[d.pc:])
	d.pc += 4
	return OpZeroFill{
		ByteLen: byteLen,
	}
}
func (d *OpDecoder) DecodePrepareFrameData() OpPrepareFrameData {
	progID := binary.LittleEndian.Uint32(d.opBuf[d.pc:])
	dataByteLen := binary.LittleEndian.Uint32(d.opBuf[d.pc+4:])
	typeID := binary.LittleEndian.Uint32(d.opBuf[d.pc+8:])
	d.pc += 12
	return OpPrepareFrameData{
		ProgID:      progID,
		DataByteLen: dataByteLen,
		TypeID:      typeID,
	}
}

type Op struct {
	Pc   int32
	Code OpCode
	Op   any
}

func (d *Op) String() string {
	return fmt.Sprintf("Op{Pc: %d, Code: %s, Op: %v}", d.Pc, d.Code, d.Op)
}

func (d *OpDecoder) PeekOp() Op {
	pc := d.pc
	defer func() { d.pc = pc }()

	code := OpCode(d.opBuf[d.pc])
	var op any
	switch code {

	case OpCodeCall:
		op = d.DecodeCall()

	case OpCodeCondJump:
		op = d.DecodeCondJump()

	case OpCodeDecrement:
		op = d.DecodeDecrement()

	case OpCodeEnqueueEmptyInterface:
		op = d.DecodeEnqueueEmptyInterface()

	case OpCodeEnqueueInterface:
		op = d.DecodeEnqueueInterface()

	case OpCodeEnqueuePointer:
		op = d.DecodeEnqueuePointer()

	case OpCodeEnqueueSliceHeader:
		op = d.DecodeEnqueueSliceHeader()

	case OpCodeEnqueueStringHeader:
		op = d.DecodeEnqueueStringHeader()

	case OpCodeEnqueueMapHeader:
		op = d.DecodeEnqueueMapHeader()

	case OpCodeJump:
		op = d.DecodeJump()

	case OpCodePop:
		op = d.DecodePop()

	case OpCodePushImm:
		op = d.DecodePushImm()

	case OpCodePushOffset:
		op = d.DecodePushOffset()

	case OpCodePushSliceLen:
		op = d.DecodePushSliceLen()

	case OpCodeReturn:
		op = d.DecodeReturn()

	case OpCodeSetOffset:
		op = d.DecodeSetOffset()

	case OpCodeShiftOffset:
		op = d.DecodeShiftOffset()

	case OpCodeEnqueueBiasedPointer:
		op = d.DecodeEnqueueBiasedPointer()

	case OpCodeDereferenceCFAOffset:
		op = d.DecodeDereferenceCFAOffset()

	case OpCodeCopyFromRegister:
		op = d.DecodeCopyFromRegister()

	case OpCodeZeroFill:
		op = d.DecodeZeroFill()

	case OpCodePrepareFrameData:
		op = d.DecodePrepareFrameData()

	}
	return Op{
		Pc:   int32(pc),
		Code: code,
		Op:   op,
	}
}
