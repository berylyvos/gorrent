package bencode

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func getNumDigit(num int) int {
	digit := 0
	if num < 0 {
		digit = 1
	}
	for {
		num /= 10
		digit++
		if num == 0 {
			break
		}
	}
	return digit
}

func TestEncDecString(t *testing.T) {
	str := "test string"
	buf := new(bytes.Buffer)

	wLen := EncodeString(buf, str)
	assert.Equal(t, len(str)+1+getNumDigit(len(str)), wLen)

	dStr, _ := DecodeString(buf)
	assert.Equal(t, str, dStr)

	str = ""
	for i := 0; i < 100; i++ {
		str += string(byte('a'))
	}
	buf.Reset()
	wLen = EncodeString(buf, str)
	assert.Equal(t, len(str)+1+getNumDigit(len(str)), wLen)
	dStr, _ = DecodeString(buf)
	assert.Equal(t, str, dStr)
}

func TestEncDecInt(t *testing.T) {
	val := 999
	buf := new(bytes.Buffer)
	wLen := EncodeInt(buf, val)
	assert.Equal(t, getNumDigit(val)+2, wLen)
	iv, _ := DecodeInt(buf)
	assert.Equal(t, val, iv)

	val = 0
	buf.Reset()
	wLen = EncodeInt(buf, val)
	assert.Equal(t, getNumDigit(val)+2, wLen)
	iv, _ = DecodeInt(buf)
	assert.Equal(t, val, iv)

	val = -99
	buf.Reset()
	wLen = EncodeInt(buf, val)
	assert.Equal(t, getNumDigit(val)+2, wLen)
	iv, _ = DecodeInt(buf)
	assert.Equal(t, val, iv)
}

func objAssertStr(t *testing.T, expect string, o *BObject) {
	assert.Equal(t, BSTR, o.typ_)
	str, err := o.Str()
	assert.Equal(t, nil, err)
	assert.Equal(t, expect, str)
}

func objAssertInt(t *testing.T, expect int, o *BObject) {
	assert.Equal(t, BINT, o.typ_)
	val, err := o.Int()
	assert.Equal(t, nil, err)
	assert.Equal(t, expect, val)
}

func TestParseStringBObject(t *testing.T) {
	var o *BObject
	in := "11:Hello World"
	buf := bytes.NewBufferString(in)
	o, _ = Parse(buf)
	objAssertStr(t, "Hello World", o)

	out := bytes.NewBufferString("")
	assert.Equal(t, len(in), o.Bencode(out))
	assert.Equal(t, in, out.String())
}

func TestParseIntBObject(t *testing.T) {
	var o *BObject
	in := "i2147483648e"
	buf := bytes.NewBufferString(in)
	o, _ = Parse(buf)
	objAssertInt(t, 2147483648, o)

	out := bytes.NewBufferString("")
	assert.Equal(t, len(in), o.Bencode(out))
	assert.Equal(t, in, out.String())
}

func TestParseList(t *testing.T) {
	var o *BObject
	var list []*BObject
	in := "li123e6:archeri789ee"
	buf := bytes.NewBufferString(in)
	o, _ = Parse(buf)
	assert.Equal(t, BLIST, o.typ_)
	list, err := o.List()
	assert.Equal(t, nil, err)
	assert.Equal(t, 3, len(list))
	objAssertInt(t, 123, list[0])
	objAssertStr(t, "archer", list[1])
	objAssertInt(t, 789, list[2])

	out := bytes.NewBufferString("")
	assert.Equal(t, len(in), o.Bencode(out))
	assert.Equal(t, in, out.String())
}

func TestParseMap(t *testing.T) {
	var o *BObject
	var dict map[string]*BObject
	in := "d4:name6:archer3:agei29ee"
	buf := bytes.NewBufferString(in)
	o, _ = Parse(buf)
	assert.Equal(t, BDICT, o.typ_)
	dict, err := o.Dict()
	assert.Equal(t, nil, err)
	objAssertStr(t, "archer", dict["name"])
	objAssertInt(t, 29, dict["age"])

	out := bytes.NewBufferString("")
	assert.Equal(t, len(in), o.Bencode(out))
}

func TestParseComMap(t *testing.T) {
	var o *BObject
	var dict map[string]*BObject
	in := "d4:userd4:name6:archer3:agei29ee5:valueli80ei85ei90eee"
	buf := bytes.NewBufferString(in)
	o, _ = Parse(buf)
	assert.Equal(t, BDICT, o.typ_)
	dict, err := o.Dict()
	assert.Equal(t, nil, err)
	assert.Equal(t, BDICT, dict["user"].typ_)
	assert.Equal(t, BLIST, dict["value"].typ_)
}
