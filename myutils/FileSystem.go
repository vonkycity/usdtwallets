package myutils

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

//GetCurrentPath 返回当前路径
func GetCurrentPath() string {
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		return ""
	}
	path, err := filepath.Abs(file)
	if err != nil {
		return ""
	}
	if runtime.GOOS == "windows" {
		path = strings.Replace(path, "\\", "/", -1)
	}
	i := strings.LastIndex(path, "/")
	if i < 0 {
		return ""
	}
	return string(path[0 : i+1])
}

//IsFileExist 判断文件是否存在
func IsFileExist(path string) bool {
	fileInfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	if fileInfo.IsDir() {
		return true
	}
	if err == nil {
		return true
	}
	return false
}
