/*
Copyright (c) 2009 The Go Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

// Type conversions tests for Scan.
// These functions are copied from database/sql/convert.go build 1.18

package opt

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"runtime"
	"testing"
	"time"
)

var someTime = time.Unix(123, 0)
var answer int64 = 42

type userDefined float64

type userDefinedSlice []int

type conversionTest struct {
	s, d any // source and destination

	// following are used if they're non-zero
	wantint    int64
	wantuint   uint64
	wantstr    string
	wantbytes  []byte
	wantraw    sql.RawBytes
	wantf32    float32
	wantf64    float64
	wanttime   time.Time
	wantbool   bool // used if d is of type *bool
	wanterr    string
	wantiface  any
	wantptr    *int64 // if non-nil, *d's pointed value must be equal to *wantptr
	wantnil    bool   // if true, *d must be *int64(nil)
	wantusrdef userDefined
}

// Target variables for scanning into.
var (
	scanstr    string
	scanbytes  []byte
	scanraw    sql.RawBytes
	scanint    int
	scanint8   int8
	scanint16  int16
	scanint32  int32
	scanuint8  uint8
	scanuint16 uint16
	scanbool   bool
	scanf32    float32
	scanf64    float64
	scantime   time.Time
	scanptr    *int64
	scaniface  any
)

var conversionTests = []conversionTest{
	// Exact conversions (destination pointer type matches source type)
	{s: "foo", d: &scanstr, wantstr: "foo"},
	{s: 123, d: &scanint, wantint: 123},
	{s: someTime, d: &scantime, wanttime: someTime},

	// To strings
	{s: "string", d: &scanstr, wantstr: "string"},
	{s: []byte("byteslice"), d: &scanstr, wantstr: "byteslice"},
	{s: 123, d: &scanstr, wantstr: "123"},
	{s: int8(123), d: &scanstr, wantstr: "123"},
	{s: int64(123), d: &scanstr, wantstr: "123"},
	{s: uint8(123), d: &scanstr, wantstr: "123"},
	{s: uint16(123), d: &scanstr, wantstr: "123"},
	{s: uint32(123), d: &scanstr, wantstr: "123"},
	{s: uint64(123), d: &scanstr, wantstr: "123"},
	{s: 1.5, d: &scanstr, wantstr: "1.5"},

	// From time.Time:
	{s: time.Unix(1, 0).UTC(), d: &scanstr, wantstr: "1970-01-01T00:00:01Z"},
	{s: time.Unix(1453874597, 0).In(time.FixedZone("here", -3600*8)), d: &scanstr, wantstr: "2016-01-26T22:03:17-08:00"},
	{s: time.Unix(1, 2).UTC(), d: &scanstr, wantstr: "1970-01-01T00:00:01.000000002Z"},
	{s: time.Time{}, d: &scanstr, wantstr: "0001-01-01T00:00:00Z"},
	{s: time.Unix(1, 2).UTC(), d: &scanbytes, wantbytes: []byte("1970-01-01T00:00:01.000000002Z")},
	{s: time.Unix(1, 2).UTC(), d: &scaniface, wantiface: time.Unix(1, 2).UTC()},

	// To []byte
	{s: nil, d: &scanbytes, wantbytes: nil},
	{s: "string", d: &scanbytes, wantbytes: []byte("string")},
	{s: []byte("byteslice"), d: &scanbytes, wantbytes: []byte("byteslice")},
	{s: 123, d: &scanbytes, wantbytes: []byte("123")},
	{s: int8(123), d: &scanbytes, wantbytes: []byte("123")},
	{s: int64(123), d: &scanbytes, wantbytes: []byte("123")},
	{s: uint8(123), d: &scanbytes, wantbytes: []byte("123")},
	{s: uint16(123), d: &scanbytes, wantbytes: []byte("123")},
	{s: uint32(123), d: &scanbytes, wantbytes: []byte("123")},
	{s: uint64(123), d: &scanbytes, wantbytes: []byte("123")},
	{s: 1.5, d: &scanbytes, wantbytes: []byte("1.5")},

	// To sql.RawBytes
	{s: nil, d: &scanraw, wantraw: nil},
	{s: []byte("byteslice"), d: &scanraw, wantraw: sql.RawBytes("byteslice")},
	{s: 123, d: &scanraw, wantraw: sql.RawBytes("123")},
	{s: int8(123), d: &scanraw, wantraw: sql.RawBytes("123")},
	{s: int64(123), d: &scanraw, wantraw: sql.RawBytes("123")},
	{s: uint8(123), d: &scanraw, wantraw: sql.RawBytes("123")},
	{s: uint16(123), d: &scanraw, wantraw: sql.RawBytes("123")},
	{s: uint32(123), d: &scanraw, wantraw: sql.RawBytes("123")},
	{s: uint64(123), d: &scanraw, wantraw: sql.RawBytes("123")},
	{s: 1.5, d: &scanraw, wantraw: sql.RawBytes("1.5")},

	// Strings to integers
	{s: "127", d: &scanint8, wantint: 127},
	{s: "128", d: &scanint8, wanterr: `converting driver.Value type string ("128") to a int8: value out of range`},
	{s: "32767", d: &scanint16, wantint: 32767},
	{s: "32768", d: &scanint16, wanterr: `converting driver.Value type string ("32768") to a int16: value out of range`},
	{s: "2147483647", d: &scanint32, wantint: 2147483647},
	{s: "2147483648", d: &scanint32, wanterr: `converting driver.Value type string ("2147483648") to a int32: value out of range`},
	{s: "255", d: &scanuint8, wantuint: 255},
	{s: "256", d: &scanuint8, wanterr: `converting driver.Value type string ("256") to a uint8: value out of range`},
	{s: "256", d: &scanuint16, wantuint: 256},
	{s: "-1", d: &scanint, wantint: -1},
	{s: "foo", d: &scanint, wanterr: `converting driver.Value type string ("foo") to a int: invalid syntax`},

	// int64 to smaller integers
	{s: int64(5), d: &scanuint8, wantuint: 5},
	{s: int64(256), d: &scanuint8, wanterr: `converting driver.Value type int64 ("256") to a uint8: value out of range`},
	{s: int64(256), d: &scanuint16, wantuint: 256},
	{s: int64(65536), d: &scanuint16, wanterr: `converting driver.Value type int64 ("65536") to a uint16: value out of range`},

	// True bools
	{s: true, d: &scanbool, wantbool: true},
	{s: "True", d: &scanbool, wantbool: true},
	{s: "TRUE", d: &scanbool, wantbool: true},
	{s: "1", d: &scanbool, wantbool: true},
	{s: 1, d: &scanbool, wantbool: true},
	{s: int64(1), d: &scanbool, wantbool: true},
	{s: uint16(1), d: &scanbool, wantbool: true},

	// False bools
	{s: false, d: &scanbool, wantbool: false},
	{s: "false", d: &scanbool, wantbool: false},
	{s: "FALSE", d: &scanbool, wantbool: false},
	{s: "0", d: &scanbool, wantbool: false},
	{s: 0, d: &scanbool, wantbool: false},
	{s: int64(0), d: &scanbool, wantbool: false},
	{s: uint16(0), d: &scanbool, wantbool: false},

	// Not bools
	{s: "yup", d: &scanbool, wanterr: `sql/driver: couldn't convert "yup" into type bool`},
	{s: 2, d: &scanbool, wanterr: `sql/driver: couldn't convert 2 into type bool`},

	// Floats
	{s: float64(1.5), d: &scanf64, wantf64: float64(1.5)},
	{s: int64(1), d: &scanf64, wantf64: float64(1)},
	{s: float64(1.5), d: &scanf32, wantf32: float32(1.5)},
	{s: "1.5", d: &scanf32, wantf32: float32(1.5)},
	{s: "1.5", d: &scanf64, wantf64: float64(1.5)},

	// Pointers
	{s: any(nil), d: &scanptr, wantnil: true},
	{s: int64(42), d: &scanptr, wantptr: &answer},

	// To any
	{s: float64(1.5), d: &scaniface, wantiface: float64(1.5)},
	{s: int64(1), d: &scaniface, wantiface: int64(1)},
	{s: "str", d: &scaniface, wantiface: "str"},
	{s: []byte("byteslice"), d: &scaniface, wantiface: []byte("byteslice")},
	{s: true, d: &scaniface, wantiface: true},
	{s: nil, d: &scaniface},
	{s: []byte(nil), d: &scaniface, wantiface: []byte(nil)},

	// To a user-defined type
	{s: 1.5, d: new(userDefined), wantusrdef: 1.5},
	{s: int64(123), d: new(userDefined), wantusrdef: 123},
	{s: "1.5", d: new(userDefined), wantusrdef: 1.5},
	{s: []byte{1, 2, 3}, d: new(userDefinedSlice), wanterr: `unsupported Scan, storing driver.Value type []uint8 into type *opt.userDefinedSlice`},

	// Other errors
	{s: complex(1, 2), d: &scanstr, wanterr: `unsupported Scan, storing driver.Value type complex128 into type *string`},
}

func intPtrValue(intptr any) any {
	return reflect.Indirect(reflect.Indirect(reflect.ValueOf(intptr))).Int()
}

func intValue(intptr any) int64 {
	return reflect.Indirect(reflect.ValueOf(intptr)).Int()
}

func uintValue(intptr any) uint64 {
	return reflect.Indirect(reflect.ValueOf(intptr)).Uint()
}

func float64Value(ptr any) float64 {
	return *(ptr.(*float64))
}

func float32Value(ptr any) float32 {
	return *(ptr.(*float32))
}

func getTimeValue(ptr any) time.Time {
	return *(ptr.(*time.Time))
}

func TestConversions(t *testing.T) {
	for n, ct := range conversionTests {
		err := ConvertAssign(ct.d, ct.s)
		errstr := ""
		if err != nil {
			errstr = err.Error()
		}
		errf := func(format string, args ...any) {
			base := fmt.Sprintf("ConvertAssign #%d: for %v (%T) -> %T, ", n, ct.s, ct.s, ct.d)
			t.Errorf(base+format, args...)
		}
		if errstr != ct.wanterr {
			errf("got error %q, want error %q", errstr, ct.wanterr)
		}
		if ct.wantstr != "" && ct.wantstr != scanstr {
			errf("want string %q, got %q", ct.wantstr, scanstr)
		}
		if ct.wantint != 0 && ct.wantint != intValue(ct.d) {
			errf("want int %d, got %d", ct.wantint, intValue(ct.d))
		}
		if ct.wantuint != 0 && ct.wantuint != uintValue(ct.d) {
			errf("want uint %d, got %d", ct.wantuint, uintValue(ct.d))
		}
		if ct.wantf32 != 0 && ct.wantf32 != float32Value(ct.d) {
			errf("want float32 %v, got %v", ct.wantf32, float32Value(ct.d))
		}
		if ct.wantf64 != 0 && ct.wantf64 != float64Value(ct.d) {
			errf("want float32 %v, got %v", ct.wantf64, float64Value(ct.d))
		}
		if bp, boolTest := ct.d.(*bool); boolTest && *bp != ct.wantbool && ct.wanterr == "" {
			errf("want bool %v, got %v", ct.wantbool, *bp)
		}
		if !ct.wanttime.IsZero() && !ct.wanttime.Equal(getTimeValue(ct.d)) {
			errf("want time %v, got %v", ct.wanttime, getTimeValue(ct.d))
		}
		if ct.wantnil && *ct.d.(**int64) != nil {
			errf("want nil, got %v", intPtrValue(ct.d))
		}
		if ct.wantptr != nil {
			if *ct.d.(**int64) == nil {
				errf("want pointer to %v, got nil", *ct.wantptr)
			} else if *ct.wantptr != intPtrValue(ct.d) {
				errf("want pointer to %v, got %v", *ct.wantptr, intPtrValue(ct.d))
			}
		}
		if len(ct.wantraw) != 0 {
			s := fmt.Sprintf("%v", ct.s)
			if _, ok := ct.s.([]byte); ok {
				s = fmt.Sprintf("%s", ct.s)
			}
			if s != string(ct.wantraw) {
				errf("want %q, got: %s", string(ct.wantraw), s)
			}
		}
		if len(ct.wantbytes) != 0 {
			s := fmt.Sprintf("%v", ct.s)
			if _, ok := ct.s.([]byte); ok {
				s = fmt.Sprintf("%s", ct.s)
			}
			if timeVal, ok := ct.s.(time.Time); ok {
				s = timeVal.Format(time.RFC3339Nano)
			}
			if s != string(ct.wantbytes) {
				errf("want %q, got: %s", string(ct.wantbytes), s)
			}
		}
		if ifptr, ok := ct.d.(*any); ok {
			if !reflect.DeepEqual(ct.wantiface, scaniface) {
				errf("want interface %#v, got %#v", ct.wantiface, scaniface)
				continue
			}
			if srcBytes, ok := ct.s.([]byte); ok {
				dstBytes := (*ifptr).([]byte)
				if len(srcBytes) > 0 && &dstBytes[0] == &srcBytes[0] {
					errf("copy into any didn't copy []byte data")
				}
			}
		}
		if ct.wantusrdef != 0 && ct.wantusrdef != *ct.d.(*userDefined) {
			errf("want userDefined %f, got %f", ct.wantusrdef, *ct.d.(*userDefined))
		}
	}
}

func TestNullString(t *testing.T) {
	var ns sql.NullString
	if err := ConvertAssign(&ns, []byte("foo")); err != nil {
		t.Error(err)
	}
	if !ns.Valid {
		t.Errorf("expecting not null")
	}
	if ns.String != "foo" {
		t.Errorf("expecting foo; got %q", ns.String)
	}
	if err := ConvertAssign(&ns, nil); err != nil {
		t.Error(err)
	}
	if ns.Valid {
		t.Errorf("expecting null on nil")
	}
	if ns.String != "" {
		t.Errorf("expecting blank on nil; got %q", ns.String)
	}
}

type valueConverterTest struct {
	c       driver.ValueConverter
	in, out any
	err     string
}

var valueConverterTests = []valueConverterTest{
	{driver.DefaultParameterConverter, sql.NullString{String: "hi", Valid: true}, "hi", ""},
	{driver.DefaultParameterConverter, sql.NullString{String: "", Valid: false}, nil, ""},
}

func TestValueConverters(t *testing.T) {
	for i, tt := range valueConverterTests {
		out, err := tt.c.ConvertValue(tt.in)
		goterr := ""
		if err != nil {
			goterr = err.Error()
		}
		if goterr != tt.err {
			t.Errorf("test %d: %T(%T(%v)) error = %q; want error = %q",
				i, tt.c, tt.in, tt.in, goterr, tt.err)
		}
		if tt.err != "" {
			continue
		}
		if !reflect.DeepEqual(out, tt.out) {
			t.Errorf("test %d: %T(%T(%v)) = %v (%T); want %v (%T)",
				i, tt.c, tt.in, tt.in, out, out, tt.out, tt.out)
		}
	}
}

// Tests that assigning to sql.RawBytes doesn't allocate (and also works).
func TestRawBytesAllocs(t *testing.T) {
	var tests = []struct {
		name string
		in   any
		want string
	}{
		{"uint64", uint64(12345678), "12345678"},
		{"uint32", uint32(1234), "1234"},
		{"uint16", uint16(12), "12"},
		{"uint8", uint8(1), "1"},
		{"uint", uint(123), "123"},
		{"int", int(123), "123"},
		{"int8", int8(1), "1"},
		{"int16", int16(12), "12"},
		{"int32", int32(1234), "1234"},
		{"int64", int64(12345678), "12345678"},
		{"float32", float32(1.5), "1.5"},
		{"float64", float64(64), "64"},
		{"bool", false, "false"},
	}

	buf := make(sql.RawBytes, 10)
	test := func(name string, in any, want string) {
		if err := ConvertAssign(&buf, in); err != nil {
			t.Fatalf("%s: ConvertAssign = %v", name, err)
		}
		match := len(buf) == len(want)
		if match {
			for i, b := range buf {
				if want[i] != b {
					match = false
					break
				}
			}
		}
		if !match {
			t.Fatalf("%s: got %q (len %d); want %q (len %d)", name, buf, len(buf), want, len(want))
		}
	}

	n := testing.AllocsPerRun(100, func() {
		for _, tt := range tests {
			test(tt.name, tt.in, tt.want)
		}
	})

	// The numbers below are only valid for 64-bit interface word sizes,
	// and gc. With 32-bit words there are more convT2E allocs, and
	// with gccgo, only pointers currently go in interface data.
	// So only care on amd64 gc for now.
	measureAllocs := runtime.GOARCH == "amd64" && runtime.Compiler == "gc"

	if n > 0.5 && measureAllocs {
		t.Fatalf("allocs = %v; want 0", n)
	}

	// This one involves a convT2E allocation, string -> any
	n = testing.AllocsPerRun(100, func() {
		test("string", "foo", "foo")
	})
	if n > 1.5 && measureAllocs {
		t.Fatalf("allocs = %v; want max 1", n)
	}
}
