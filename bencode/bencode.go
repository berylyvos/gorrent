package bencode

import (
	"bufio"
	"errors"
	"io"
)

type BType uint8
type BValue interface{}

const (
	BSTR BType = iota
	BINT
	BLIST
	BDICT
)

var (
	ErrNum = errors.New("expect num")
	ErrCol = errors.New("expect colon")
	ErrEpI = errors.New("expect 'i'")
	ErrEpE = errors.New("expect 'e'")
	ErrTyp = errors.New("wrong type")
	ErrIvd = errors.New("invalid bencode")
)

type BObject struct {
	typ_ BType
	val_ BValue
}

func (o *BObject) Str() (string, error) {
	if o.typ_ != BSTR {
		return "", ErrTyp
	}
	return o.val_.(string), nil
}

func (o *BObject) Int() (int, error) {
	if o.typ_ != BINT {
		return 0, ErrTyp
	}
	return o.val_.(int), nil
}

func (o *BObject) List() ([]*BObject, error) {
	if o.typ_ != BLIST {
		return nil, ErrTyp
	}
	return o.val_.([]*BObject), nil
}

func (o *BObject) Dict() (map[string]*BObject, error) {
	if o.typ_ != BDICT {
		return nil, ErrTyp
	}
	return o.val_.(map[string]*BObject), nil
}

func (o *BObject) Bencode(w io.Writer) int {
	bw, ok := w.(*bufio.Writer)
	if !ok {
		bw = bufio.NewWriter(w)
	}
	wLen := 0
	switch o.typ_ {
	case BSTR:
		str, _ := o.Str()
		wLen += EncodeString(bw, str)
	case BINT:
		num, _ := o.Int()
		wLen += EncodeInt(bw, num)
	case BLIST:
		_ = bw.WriteByte('l')
		list, _ := o.List()
		for _, obj := range list {
			wLen += obj.Bencode(bw)
		}
		_ = bw.WriteByte('e')
		wLen += 2
	case BDICT:
		_ = bw.WriteByte('d')
		dict, _ := o.Dict()
		for key, obj := range dict {
			wLen += EncodeString(bw, key)
			wLen += obj.Bencode(bw)
		}
		_ = bw.WriteByte('e')
		wLen += 2
	}
	bw.Flush()
	return wLen
}

func checkNum(data byte) bool {
	return data >= '0' && data <= '9'
}

func readDecimal(r *bufio.Reader) (val, len int) {
	sign := 1
	b, _ := r.ReadByte()
	len++
	if b == '-' {
		sign = -1
		b, _ = r.ReadByte()
		len++
	}
	for {
		if !checkNum(b) {
			_ = r.UnreadByte()
			len--
			return sign * val, len
		}
		val = val*10 + int(b-'0')
		b, _ = r.ReadByte()
		len++
	}
}

func writeDecimal(w *bufio.Writer, val int) (len int) {
	if val == 0 {
		_ = w.WriteByte('0')
		len++
		return
	}
	if val < 0 {
		_ = w.WriteByte('-')
		len++
		val *= -1
	}

	dividend := 1
	for {
		if dividend > val {
			dividend /= 10
			break
		}
		dividend *= 10
	}
	for {
		_ = w.WriteByte(byte(val/dividend) + '0')
		len++
		if dividend == 1 {
			return
		}
		val %= dividend
		dividend /= 10
	}
}

func EncodeString(w io.Writer, val string) int {
	strLen := len(val)
	bw := bufio.NewWriter(w)
	wLen := writeDecimal(bw, strLen)
	_ = bw.WriteByte(':')
	wLen++
	_, _ = bw.WriteString(val)
	wLen += strLen

	err := bw.Flush()
	if err != nil {
		return 0
	}
	return wLen
}

func DecodeString(r io.Reader) (string, error) {
	str := ""
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}
	strLen, nLen := readDecimal(br)
	if nLen == 0 {
		return str, ErrNum
	}
	b, err := br.ReadByte()
	if b != ':' {
		return str, ErrCol
	}
	buf := make([]byte, strLen)
	_, err = io.ReadAtLeast(br, buf, strLen)
	str = string(buf)
	return str, err
}

func EncodeInt(w io.Writer, val int) int {
	bw := bufio.NewWriter(w)
	wLen := 0
	_ = bw.WriteByte('i')
	wLen++
	nLen := writeDecimal(bw, val)
	wLen += nLen
	_ = bw.WriteByte('e')
	wLen++

	err := bw.Flush()
	if err != nil {
		return 0
	}
	return wLen
}

func DecodeInt(r io.Reader) (int, error) {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}
	b, err := br.ReadByte()
	if b != 'i' {
		return 0, ErrEpI
	}
	val, _ := readDecimal(br)
	b, err = br.ReadByte()
	if b != 'e' {
		return val, ErrEpE
	}
	return val, err
}
