package myutils

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
)

//EnableDebug 是否打开调试模式
var EnableDebug bool

var anyMap = map[int64]string{
	0:  "0",
	1:  "1",
	2:  "2",
	3:  "3",
	4:  "4",
	5:  "5",
	6:  "6",
	7:  "7",
	8:  "8",
	9:  "9",
	10: "a",
	11: "b",
	12: "c",
	13: "d",
	14: "e",
	15: "f",
	16: "g",
	17: "h",
	18: "i",
	19: "j",
	20: "k",
	21: "l",
	22: "m",
	23: "n",
	24: "o",
	25: "p",
	26: "q",
	27: "r",
	28: "s",
	29: "t",
	30: "u",
	31: "v",
	32: "w",
	33: "x",
	34: "y",
	35: "z",
	36: "A",
	37: "B",
	38: "C",
	39: "D",
	40: "E",
	41: "F",
	42: "G",
	43: "H",
	44: "I",
	45: "J",
	46: "K",
	47: "L",
	48: "M",
	49: "N",
	50: "O",
	51: "P",
	52: "Q",
	53: "R",
	54: "S",
	55: "T",
	56: "U",
	57: "V",
	58: "W",
	59: "X",
	60: "Y",
	61: "Z",
}

func findKey(in string) int64 {
	var result int64
	result = -1
	for k, v := range anyMap {
		if in == v {
			result = k
		}
	}
	return result
}

// Println 打印内容
func Println(contents ...interface{}) {
	if EnableDebug {
		log.Println(contents...)
	}
}

// Print 打印内容
func Print(contents ...interface{}) {
	if EnableDebug {
		log.Print(contents...)
	}
}

//DecToAny 十进制转任何进制，效率低
func DecToAny(num int64, n int64) string {
	if n <= 1 {
		return ""
	}
	newNumStr := ""
	var remainder int64
	var remainderString string
	for num != 0 {
		remainder = num % n
		if 76 > remainder && remainder > 9 {
			remainderString = anyMap[remainder]
		} else {
			//remainderString = strconv.Itoa(remainder)
			remainderString = strconv.FormatInt(remainder, 10)
		}
		newNumStr = remainderString + newNumStr
		num = num / n
	}
	return newNumStr
}

//AnyToDec 任意进制转十进制
func AnyToDec(num string, n int) int64 {
	// ex
	// fmt.Println(decToAny(1, 62))
	// fmt.Println(anyToDec("1", 62))
	var newNum float64
	newNum = 0.0
	nNum := len(strings.Split(num, "")) - 1
	for _, value := range strings.Split(num, "") {
		tmp := float64(findKey(value))
		if tmp != -1 {
			newNum = newNum + tmp*math.Pow(float64(n), float64(nNum))
			nNum = nNum - 1
		} else {
			break
		}
	}
	return int64(newNum)
	//return int(newNum)
}

//TypeOf 返回变量类型
func TypeOf(v interface{}) string {
	return fmt.Sprintf("%T", v)
}

//SafeWriteStringChannel 安全写入channel 不会panic
func SafeWriteStringChannel(writeChan chan<- string, message string) (closed bool) {
	defer func() {
		if recover() != nil {
			closed = true
		}
	}()
	if writeChan == nil {
		return true
	}
	writeChan <- message
	return false
}

//SafeWriteBoolChannel 安全写入channel
func SafeWriteBoolChannel(boolChan chan<- bool, v bool) (closed bool) {
	defer func() {
		if recover() != nil {
			closed = true
		}
	}()
	if boolChan == nil {
		return true
	}
	boolChan <- v
	return false
}

//SafeCloseStringChannel //安全关闭channel
func SafeCloseStringChannel(ch chan string) (closed bool) {
	defer func() {
		if recover() != nil {
			closed = false
		}
	}()
	if ch == nil {
		return true
	}
	close(ch)
	return true
}

//DecToBin 十进制转二进制
func DecToBin(n int) string {
	if n < 0 {
		return ""
	}
	if n == 0 {
		return "0"
	}
	s := ""
	for q := n; q > 0; q = q / 2 {
		m := q % 2
		s = fmt.Sprintf("%v%v", m, s)
	}
	return s
}

//DecToOct 十进制转八进制
func DecToOct(d int) int {
	if d == 0 {
		return 0
	}
	if d < 0 {
		return -1
	}
	s := ""
	for q := d; q > 0; q = q / 8 {
		m := q % 8
		s = fmt.Sprintf("%v%v", m, s)
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return int(n)
}

//DecToHex 十进制转十六进制
func DecToHex(n int) string {
	if n < 0 {
		return ""
	}
	if n == 0 {
		return "0"
	}
	hex := map[int]int{10: 65, 11: 66, 12: 67, 13: 68, 14: 69, 15: 70}
	s := ""
	for q := n; q > 0; q = q / 16 {
		m := q % 16
		if m > 9 && m < 16 {
			m = hex[m]
			s = fmt.Sprintf("%v%v", string(m), s)
			continue
		}
		s = fmt.Sprintf("%v%v", m, s)
	}
	return s
}

//BinToDec 二进制转十进制
func BinToDec(b string) int {
	s := strings.Split(b, "")
	l := len(s)
	i := 0
	d := float64(0)
	for i = 0; i < l; i++ {
		f, err := strconv.ParseFloat(s[i], 10)
		if err != nil {
			return -1
		}
		d += f * math.Pow(2, float64(l-i-1))
	}
	return int(d)
}

//OctToDec 八进制转十进制
func OctToDec(o int) int {
	s := strings.Split(strconv.Itoa(int(o)), "")
	l := len(s)
	i := 0
	d := float64(0)
	for i = 0; i < l; i++ {
		f, err := strconv.ParseFloat(s[i], 10)
		if err != nil {
			return -1
		}
		d += f * math.Pow(8, float64(l-i-1))
	}
	return int(d)
}

//HexToDec 十六进制转十进制
func HexToDec(h string) int {
	s := strings.Split(strings.ToUpper(h), "")
	l := len(s)
	//i := 0
	d := float64(0)
	hex := map[string]string{"A": "10", "B": "11", "C": "12", "D": "13", "E": "14", "F": "15"}
	for i := 0; i < l; i++ {
		c := s[i]
		if v, ok := hex[c]; ok {
			c = v
		}
		f, err := strconv.ParseFloat(c, 10)
		if err != nil {
			return -1
		}
		d += f * math.Pow(16, float64(l-i-1))
	}
	return int(d)
}

//OctToBin 八进制转二进制
func OctToBin(o int) string {
	d := OctToDec(o)
	if d == -1 {
		return ""
	}
	return DecToBin(d)
}

//HexToBin 十六进制转二进制
func HexToBin(h string) string {
	d := HexToDec(h)
	if d == -1 {
		return ""
	}
	return DecToBin(d)
}

//BinToOct 二进制转八进制
func BinToOct(b string) int {
	d := BinToDec(b)
	if d == -1 {
		return -1
	}
	return DecToOct(d)
}

//BinToHex 二进制转十六进制
func BinToHex(b string) string {
	d := BinToDec(b)
	if d == -1 {
		return ""
	}
	return DecToHex(d)
}
