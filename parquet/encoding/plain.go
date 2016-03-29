package encoding

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/kostya-sh/parquet-go/parquet/encoding/rle"
	"github.com/kostya-sh/parquet-go/parquet/thrift"
)

// Decoder interface
type Decoder interface {
	DecodeBool([]bool) (count uint, err error)
	DecodeInt32([]int32) (count uint, err error)
	DecodeInt64([]int64) (count uint, err error)
	//	DecodeInt96([]int64, []int32) (count uint, err error)
	//DecodeString([]string) (count uint, err error)
	DecodeByteArray([][]byte) (count uint, err error)
	DecodeFloat32([]float32) (count uint, err error)
	DecodeFloat64([]float64) (count uint, err error)
}

func trailingZeros(i uint32) uint32 {
	var count uint32

	mask := uint32(1 << 31)
	for mask&i != mask {
		mask >>= 1
		count++
	}
	return count
}

func GetBitWidthFromMaxInt(i uint32) uint {
	return uint(32 - trailingZeros(i))
}

func min(a, b uint) uint {
	if a > b {
		return b
	}
	return a
}

// Plain
type plainDecoder struct {
	r     io.Reader
	count uint
}

// NewPlainDecoder creates a new Decoder that uses the PLAIN=0 encoding
func NewPlainDecoder(r io.Reader, numValues uint) Decoder {
	return &plainDecoder{r, numValues}
}

// DecodeBool
func (d *plainDecoder) DecodeBool(out []bool) (uint, error) {
	dec := rle.NewHybridBitPackingRLEDecoder(d.r)
	outx := make([]uint64, 0, d.count)
	err := dec.Read(outx, 1)
	if err != nil {
		return 0, err
	}

	for i := 0; i < len(outx) && i < len(out); i++ {
		out[i] = outx[i] != 0
	}

	return uint(len(outx)), nil
}

// DecodeInt32
func (d *plainDecoder) DecodeInt32(out []int32) (uint, error) {
	count := d.count

	for i := uint(0); i < count; i++ {
		var value int32
		err := binary.Read(d.r, binary.LittleEndian, &value)
		if err != nil {
			return i, fmt.Errorf("expected %d int32 but got only %d: %s", count, i, err) // FIXME
		}

		out[i] = value
	}

	return count, nil
}

// DecodeInt64
func (d *plainDecoder) DecodeInt64(out []int64) (uint, error) {
	count := d.count
	var value int64

	for i := uint(0); i < min(count, uint(len(out))); i++ {
		err := binary.Read(d.r, binary.LittleEndian, &value)
		if err != nil {
			return i, fmt.Errorf("expected %d int64 but got only %d: %s", count, i, err) // FIXME
		}

		out[i] = value
	}

	return count, nil
}

// DecodeStr , returns the number of element read, or error
func (d *plainDecoder) DecodeString(out []string) (uint, error) {
	count := d.count

	var size int32

	for i := uint(0); i < min(count, uint(len(out))); i++ {
		err := binary.Read(d.r, binary.LittleEndian, &size)
		if err != nil {
			return 0, err
		}
		p := make([]byte, size)
		n, err := d.r.Read(p)
		if err != nil {
			return i, fmt.Errorf("plain decoder: short read: %s", err)
		}

		out[i] = string(p[:n])
	}

	return count, nil
}

// DecodeStr , returns the number of element read, or error
func (d *plainDecoder) DecodeByteArray(out [][]byte) (uint, error) {
	count := d.count

	var size int32

	for i := uint(0); i < min(count, uint(len(out))); i++ {
		err := binary.Read(d.r, binary.LittleEndian, &size)
		if err != nil {
			return 0, err
		}
		p := make([]byte, size)
		n, err := d.r.Read(p)
		if err != nil {
			return i, fmt.Errorf("plain decoder: short read: %s", err)
		}
		out[i] = p[:n]
	}

	return count, nil
}

// DecodeFloat32 returns the number of elements read, or error
// The data has to be 4 bytes IEEE little endian back to back
func (d *plainDecoder) DecodeFloat32(out []float32) (uint, error) {
	count := d.count

	var value float32

	for i := uint(0); i < min(count, uint(len(out))); i++ {
		err := binary.Read(d.r, binary.LittleEndian, &value)
		if err != nil {
			return i, fmt.Errorf("plain decoder: binary.Read: %s", err)
		}

		out[i] = value
	}

	return count, nil
}

// DecodeFloat64 returns the number of elements read, or error
// The data has to be 8 bytes IEEE little endian back to back
func (d *plainDecoder) DecodeFloat64(out []float64) (uint, error) {
	count := d.count

	var value float64

	for i := uint(0); i < min(count, uint(len(out))); i++ {
		err := binary.Read(d.r, binary.LittleEndian, &value)
		if err != nil {
			return 0, fmt.Errorf("plain decoder: binary.Read: %s", err)
		}
		out[i] = value
	}

	return count, nil
}

// plain Encoder
type plainEncoder struct {
	numValues int
}

// NewPlainEncoder creates an encoder that uses the Plain encoding to store data
// inside a DataPage
func NewPlainEncoder() Encoder {
	return &plainEncoder{}
}

func (p *plainEncoder) Flush() error {
	return nil
}

func (p *plainEncoder) NumValues() int {
	return p.numValues
}

func (p *plainEncoder) Type() thrift.Encoding {
	return thrift.Encoding_PLAIN
}

// WriteBool
func (e *plainEncoder) WriteBool(w io.Writer, v []bool) error {
	e.numValues += len(v)
	return binary.Write(w, binary.LittleEndian, v)
}

// WriteInt32
func (e *plainEncoder) WriteInt32(w io.Writer, v []int32) error {
	e.numValues += len(v)
	return binary.Write(w, binary.LittleEndian, v)
}

// WriteInt64
func (e *plainEncoder) WriteInt64(w io.Writer, v []int64) error {
	e.numValues += len(v)
	return binary.Write(w, binary.LittleEndian, v)
}

// WriteFloat32
func (e *plainEncoder) WriteFloat32(w io.Writer, v []float32) error {
	e.numValues += len(v)
	return binary.Write(w, binary.LittleEndian, v)
}

// WriteFloat64
func (e *plainEncoder) WriteFloat64(w io.Writer, v []float64) error {
	e.numValues += len(v)
	return binary.Write(w, binary.LittleEndian, v)
}

// WriteByteArray
func (e *plainEncoder) WriteByteArray(w io.Writer, v [][]byte) error {
	e.numValues += len(v)
	for _, b := range v {
		err := binary.Write(w, binary.LittleEndian, len(b))
		if err != nil {
			return fmt.Errorf("could not write byte array len: %s", err)
		}
		err = binary.Write(w, binary.LittleEndian, b)
		if err != nil {
			return fmt.Errorf("could not write byte array: %s", err)
		}
	}

	return nil
}
