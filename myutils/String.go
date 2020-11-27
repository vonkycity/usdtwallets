package myutils

import (
	"crypto/md5"
	"encoding/hex"
	"math/rand"
	"regexp"
	"time"

	uuid "github.com/satori/go.uuid"
)

// GetMd5 返回md5
func GetMd5(text string) string {
	ctx := md5.New()
	ctx.Write([]byte(text))
	return hex.EncodeToString(ctx.Sum(nil))
}

//GenUUID 获取uuid
func GenUUID() string {
	myUUID := uuid.NewV4()
	return myUUID.String()
}

//GenRandomString 生成随机字符串
func GenRandomString(lenth int) string {
	rand.Seed(time.Now().UnixNano())
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ~@#$^&()-=_+/.,><|[]{}"
	b := make([]byte, lenth)
	for i := range b {
		b[i] = str[rand.Int63()%int64(len(str))]
	}
	return string(b)
}

//GenRandomNumber 生成随机字符串(数字)
func GenRandomNumber(lenth int) string {
	rand.Seed(time.Now().UnixNano())
	str := "0123456789"
	b := make([]byte, lenth)
	for i := range b {
		b[i] = str[rand.Int63()%int64(len(str))]
	}
	return string(b)
}

//GenRandNumberArray 生成count个[start,end)结束的不重复的随机数
func GenRandNumberArray(start int, end int, count int) []int {
	if end < start || (end-start) < count {
		return nil
	}
	nums := make([]int, 0)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for len(nums) < count {
		num := r.Intn((end - start)) + start
		exist := false
		for _, v := range nums {
			if v == num {
				exist = true
				break
			}
		}
		if !exist {
			nums = append(nums, num)
		}
	}
	return nums
}

// IsEmail 检查是否email
func IsEmail(word string) bool {
	//re := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	re := regexp.MustCompile(`^[A-Za-z\d]+([-_.][A-Za-z\d]+)*@([A-Za-z\d]+[-.])+[A-Za-z\d]{2,4}$`)
	if re.MatchString(word) {
		return true
	}
	return false
}

// IsMobile 检查是否mobile
func IsMobile(word string) bool {
	//re := regexp.MustCompile(`^?((?:\?[\-\.\ \\\/]?){0,})(?:[\-\.\ \\\/]?[\-\.\ \\\/]?(\d+))?$`)
	re := regexp.MustCompile(`^1[356789]\d{9}$`)
	if re.MatchString(word) {
		return true
	}
	return false
}
