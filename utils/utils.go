package utils

import (
	"os"
	"testing"
)

// WithTempDir creates a temp directory, changes the current working directory
// to it, and returns a function that can be called to clean up. Use it like
// 创建一个临时目录，并将当前工作目录切换到该临时目录，然后返回一个可以被调用用来清理它的函数。
// this:
//
//	defer WithTempDir(t)()
func WithTempDir(t *testing.T) func() {
	// 返回当前运行对应的根目录
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("TempDir: %v", err)
	}

	//创建一个临时目录并返回目录名称，清楚创建的临时目录是调用者应该做的
	//tmpdir, err := ioutil.TempDir("", "")
	tmpdir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatalf("TempDir: %v", err)
	}
	//切换当前工作目录到临时目录
	err = os.Chdir(tmpdir)
	if err != nil {
		t.Fatalf("Chdir: %v", err)
	}

	//返回一个可以用来清理临时目录的函数
	return func() {
		// 闭包函数，cwd变量引用当前环境的值
		err := os.Chdir(cwd)
		if err != nil {
			t.Fatalf("Chdir: %v", err)
		}
		// 清理临时目录
		err = os.RemoveAll(tmpdir)
		if err != nil {
			t.Fatalf("Removing tmpdir: %s", err)
		}
	}
}
