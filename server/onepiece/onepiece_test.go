package onepiece

import (
	"testing"
	"reflect"
)

type(
	msg = []interface{}
	b = []byte
	ss = []string
)

const NUMITEMS = 10

var tests = []struct {
	decoded []interface{}
	encoded []byte
	structure []string //"number" or "bytearray"
}{
	{msg{123}, b("123"), ss{"number"}},
	{msg{b("123")}, b("\"123\""), ss{"bytearray"}},
	{msg{b("a\"b")}, b("\"a\\\"b\""), ss{"bytearray"}},
	{msg{b("123"), 233, b("a\nb")}, b("\"123\",233,\"a\nb\""), ss{"bytearray", "number", "bytearray"}},
}

func TestEncode(t *testing.T) {
	for _, tt := range tests {
		testname := string(tt.encoded)
		t.Run(testname, func(t *testing.T){
			encodedMsg := EncodeMsg(tt.decoded)
			if !reflect.DeepEqual(encodedMsg, tt.encoded) {
				t.Errorf("got %v; wanted %v", encodedMsg, tt.encoded)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	for _, tt := range tests {
		testname := string(tt.encoded)
		t.Run(testname, func(t *testing.T){
			numbers, bytearrays := ParseMsg(tt.encoded, NUMITEMS)
			decodedMsg := []interface{}{}
			for i, v := range tt.structure {
				if v == "bytearray" {
					decodedMsg = append(decodedMsg, GetBytearray(numbers, bytearrays, i))
				} else {
					decodedMsg = append(decodedMsg, numbers[i])
				}
			}
			if !reflect.DeepEqual(decodedMsg, tt.decoded) {
				t.Errorf("got %v; wanted %v", decodedMsg, tt.decoded)
			}
		})
	}
}
