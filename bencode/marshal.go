package bencode

import (
	"errors"
	"io"
	"reflect"
	"strings"
)

const BENCODE = "bencode"

func Marshal(w io.Writer, s interface{}) int {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return MarshalValue(w, v)
}

func MarshalValue(w io.Writer, v reflect.Value) int {
	wLen := 0
	switch v.Kind() {
	case reflect.Int:
		wLen += EncodeInt(w, int(v.Int()))
	case reflect.String:
		wLen += EncodeString(w, v.String())
	case reflect.Slice:
		wLen += marshalList(w, v)
	case reflect.Struct:
		wLen += marshalDict(w, v)
	}
	return wLen
}

func marshalDict(w io.Writer, v reflect.Value) int {
	wLen := 2
	_, _ = w.Write([]byte{'d'})
	for i, n := 0, v.NumField(); i < n; i++ {
		fv := v.Field(i)
		ft := v.Type().Field(i)
		key := ft.Tag.Get(BENCODE)
		if key == "" {
			key = strings.ToLower(ft.Name)
		}
		wLen += EncodeString(w, key)
		wLen += MarshalValue(w, fv)
	}
	_, _ = w.Write([]byte{'e'})
	return wLen
}

func marshalList(w io.Writer, v reflect.Value) int {
	wLen := 2
	_, _ = w.Write([]byte{'l'})
	for i := 0; i < v.Len(); i++ {
		wLen += MarshalValue(w, v.Index(i))
	}
	_, _ = w.Write([]byte{'e'})
	return wLen
}

func unmarshalList(p reflect.Value, list []*BObject) error {
	if p.Kind() != reflect.Ptr || p.Elem().Type().Kind() != reflect.Slice {
		return errors.New("must be pointer to slice")
	}
	v := p.Elem()
	if len(list) == 0 {
		return nil
	}
	switch list[0].typ_ {
	case BSTR:
		for i, item := range list {
			val, err := item.Str()
			if err != nil {
				return err
			}
			v.Index(i).SetString(val)
		}
	case BINT:
		for i, item := range list {
			val, err := item.Int()
			if err != nil {
				return err
			}
			v.Index(i).SetInt(int64(val))
		}
	case BLIST:
		for i, item := range list {
			val, err := item.List()
			if err != nil {
				return err
			}
			if v.Type().Elem().Kind() != reflect.Slice {
				return ErrTyp
			}
			lp := reflect.New(v.Type().Elem())
			ls := reflect.MakeSlice(v.Type().Elem(), len(val), len(val))
			lp.Elem().Set(ls)
			err = unmarshalList(lp, val)
			if err != nil {
				return err
			}
			v.Index(i).Set(lp.Elem())
		}
	case BDICT:
		for i, item := range list {
			val, err := item.Dict()
			if err != nil {
				return err
			}
			if v.Type().Elem().Kind() != reflect.Struct {
				return ErrTyp
			}
			sp := reflect.New(v.Type().Elem())
			err = unmarshalDict(sp, val)
			if err != nil {
				return err
			}
			v.Index(i).Set(sp.Elem())
		}
	}
	return nil
}

func unmarshalDict(p reflect.Value, dict map[string]*BObject) error {
	if p.Kind() != reflect.Ptr || p.Elem().Type().Kind() != reflect.Struct {
		return errors.New("must be pointer to struct")
	}
	v := p.Elem()
	for i, n := 0, v.NumField(); i < n; i++ {
		fv := v.Field(i)
		if !fv.CanSet() {
			continue
		}
		ft := v.Type().Field(i)
		key := ft.Tag.Get(BENCODE)
		if key == "" {
			key = strings.ToLower(ft.Name)
		}
		po := dict[key]
		if po == nil {
			continue
		}
		switch po.typ_ {
		case BSTR:
			if ft.Type.Kind() != reflect.String {
				return ErrTyp
			}
			val, err := po.Str()
			if err != nil {
				return err
			}
			fv.SetString(val)
		case BINT:
			if ft.Type.Kind() != reflect.Int {
				return ErrTyp
			}
			val, err := po.Int()
			if err != nil {
				return err
			}
			fv.SetInt(int64(val))
		case BLIST:
			if ft.Type.Kind() != reflect.Slice {
				return ErrTyp
			}
			val, err := po.List()
			if err != nil {
				return err
			}
			lp := reflect.New(ft.Type)
			ls := reflect.MakeSlice(ft.Type, len(val), len(val))
			lp.Elem().Set(ls)
			err = unmarshalList(lp, val)
			if err != nil {
				return err
			}
			fv.Set(lp.Elem())
		case BDICT:
			if ft.Type.Kind() != reflect.Struct {
				return ErrTyp
			}
			val, err := po.Dict()
			if err != nil {
				return err
			}
			sp := reflect.New(ft.Type)
			err = unmarshalDict(sp, val)
			if err != nil {
				return err
			}
			fv.Set(sp.Elem())
		}
	}
	return nil
}

func Unmarshal(r io.Reader, s interface{}) error {
	o, err := Parse(r)
	if err != nil {
		return err
	}
	p := reflect.ValueOf(s)
	if p.Kind() != reflect.Ptr {
		return errors.New("unmarshal destination must be a pointer")
	}
	switch o.typ_ {
	case BLIST:
		val, err := o.List()
		if err != nil {
			return err
		}
		sl := reflect.MakeSlice(p.Elem().Type(), len(val), len(val))
		p.Elem().Set(sl)
		err = unmarshalList(p, val)
		if err != nil {
			return err
		}
	case BDICT:
		val, err := o.Dict()
		if err != nil {
			return err
		}
		err = unmarshalDict(p, val)
		if err != nil {
			return err
		}
	default:
		return errors.New("unmarshal source must be slice or dict")
	}
	return nil
}
