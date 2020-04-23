package onepiece

import (
	"strconv"
	"strings"
	"fmt"
)

// leaks numFields and len(b)
// if there is a really realy long binary field full of '\"' there might be content leaks due to thje cache always staying hot
func ParseMsg(b []byte, numItems uint) ([]int, [][]byte) {
	item := 0
	inBinary := 0
	var bool2int = map[bool]int{false: 0, true: 1}
	precedingBackslash := 0
	innerItem := 0
	numbers := make([]int, numItems)
	binaries := make([][]byte, numItems)
	for i := range binaries {
		binaries[i] = make([]byte, len(b))
	}
	// there be demons
	for _, v := range b {
		changeItem := bool2int[(v==',')] & (1 - inBinary)
		numbers[item] = (changeItem * numbers[item]) + (1 - changeItem)*(inBinary*innerItem+(1-inBinary)*(numbers[item]*10+int(v)-int('0')))
		delimiter := bool2int[(v=='"')]
		backslashedDelimiter := delimiter & precedingBackslash
		innerItem -= backslashedDelimiter
		nonbackslashedDelimiter := delimiter & (1 - precedingBackslash)
		inBinary = inBinary ^ nonbackslashedDelimiter
		precedingBackslash = bool2int[(v=='\\')]
		binaries[item][innerItem] = v
		innerItem = (1-changeItem)*(innerItem+1)
		innerItem -= nonbackslashedDelimiter & inBinary
		item += changeItem
	}
	return numbers, binaries
}

func encodeBytearray(msg []byte, b []byte) []byte {
	msg = append(msg, '"')
	msg = append(msg, []byte(strings.ReplaceAll(string(b), "\"", "\\\""))...)
	msg = append(msg, '"', ',')
	return msg
}

// not branchless (leaks almost everyting)
func EncodeMsg(arr []interface{}) []byte {
	msg := make([]byte, 0)
	for i, v := range arr {
		switch v.(type){
		case string:
			msg = encodeBytearray(msg, []byte(v.(string)))
		case []byte:
			msg = encodeBytearray(msg, v.([]byte))
		case int:
			msg = append(msg, []byte(strconv.Itoa(v.(int)))...)
			msg = append(msg, ',')
		default:
			fmt.Printf("item %v has type %T", i, v)
			panic("Messages can only encode bytearrays and ints")
		}
	}
	// Assertion: none of the messages are empty
	return msg[:len(msg)-1]
}

func GetBytearray(numbers []int, bytearrays [][]byte, index int) []byte {
    return bytearrays[index][:numbers[index]]
}
