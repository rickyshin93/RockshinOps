package metrics

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestDecodeSMC(t *testing.T) {
	flt := make([]byte, 4)
	binary.LittleEndian.PutUint32(flt, math.Float32bits(41.25))
	nan := make([]byte, 4)
	binary.LittleEndian.PutUint32(nan, math.Float32bits(float32(math.NaN())))
	cases := []struct {
		name, kind string
		data       []byte
		want       float64
		ok         bool
	}{
		{"flt little endian", "flt ", flt, 41.25, true},
		{"flt NaN", "flt ", nan, 0, false},
		{"sp78", "sp78", []byte{0x1c, 0x80}, 28.5, true},
		{"sp78 negative", "sp78", []byte{0xff, 0x80}, -0.5, true},
		{"sp96", "sp96", []byte{0x02, 0x40}, 9, true},
		{"fpe2 fan", "fpe2", []byte{0x15, 0x18}, 1350, true},
		{"fp88", "fp88", []byte{0x01, 0x80}, 1.5, true},
		{"ui8", "ui8 ", []byte{2}, 2, true},
		{"ui16", "ui16", []byte{0x01, 0x02}, 258, true},
		{"ui32", "ui32", []byte{0, 1, 0, 0}, 65536, true},
		{"si8", "si8 ", []byte{0xfe}, -2, true},
		{"si16", "si16", []byte{0xff, 0xfe}, -2, true},
		{"short payload", "sp78", []byte{0x1c}, 0, false},
		{"unknown type", "ch8*", []byte{'a', 'b'}, 0, false},
	}
	for _, c := range cases {
		got, ok := decodeSMC(c.kind, c.data)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("%s: got %v %v, want %v %v", c.name, got, ok, c.want, c.ok)
		}
	}
}
