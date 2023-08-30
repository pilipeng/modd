package modd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cortesi/modd/conf"
	"github.com/cortesi/modd/utils"
	"github.com/cortesi/moddwatch"
	"github.com/cortesi/termlog"
)

const timeout = 5 * time.Second

// 修改文件内容
func touch(p string) {
	// 转换为兼容方式的目录分隔符
	p = filepath.FromSlash(p)
	// 生成的相对路径的文件
	d := filepath.Dir(p)
	err := os.MkdirAll(d, 0777)
	if err != nil {
		panic(err)
	}

	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		panic(err)
	}
	if _, err := f.Write([]byte("teststring")); err != nil {
		panic(err)
	}
	if err := f.Close(); err != nil {
		panic(err)
	}
	// ioutil 1.16后作废
	// ioutil.ReadFile(p)
	_, _ = os.ReadFile(p)
}

func events(p string) []string {
	parts := []string{}
	for _, p := range strings.Split(p, "\n") {
		if strings.HasPrefix(p, ":") {
			p = strings.TrimSpace(p)
			if !strings.HasSuffix(p, ":") {
				parts = append(parts, strings.TrimSpace(p))
			}
		}
	}
	return parts
}

// 监控 修改 排除
func _testWatch(t *testing.T, modfunc func(), expected []string) {
	// 运行完成后，清理临时目录
	defer utils.WithTempDir(t)()

	err := os.MkdirAll("a/inner", 0777)
	if err != nil {
		t.Fatal(err)
	}

	err = os.MkdirAll("b", 0777)
	if err != nil {
		t.Fatal(err)
	}

	// 初始化文件内容
	touch("a/initial")
	// There's some race condition in rjeczalik/notify. If we don't wait a bit
	// here, we sometimes receive notifications for the change above even
	// though we haven't started the watcher.
	// 休眠一会儿，否则 notify 可能会收到初始化的回调修改通知
	time.Sleep(200 * time.Millisecond)

	// 模拟配置文件内容
	confTxt := `
		@shell = bash

        ** {
            prep +onchange: echo ":skipit:" @mods
            prep: echo ":all:" @mods
        }
        a/* {
            prep: echo ":a:" @mods
        }
        b/* {
            prep: echo ":b:" @mods
        }
        a/**/*.xxx {
            prep: echo ":c:" @mods
        }
        a/direct {
            prep: echo ":d:" @mods
        }
        direct {
            prep: echo ":e:" @mods
        }
    `
	// 加载配置变量
	cnf, err := conf.Parse("test", confTxt)
	if err != nil {
		t.Fatal(err)
	}

	//创建一个日志对象
	lt := termlog.NewLogTest()
	modchan := make(chan *moddwatch.Mod, 1024)
	//回调
	cback := func() {
		start := time.Now()
		modfunc()
		for {
			/**
			17:02:42: skipping prep: echo ":skipit:" "./a/initial"
			17:02:42: prep: echo ":all:" "./a/initial"
			:all: ./a/initial
			>> done (819.0865ms)
			17:02:43: prep: echo ":a:" "./a/initial"
			:a: ./a/initial
			>> done (124.4291ms)
			17:02:43: prep: echo ":b:"
			:b:
			>> done (121.5141ms)
			17:02:43: prep: echo ":c:"
			:c:
			>> done (119.6753ms)
			17:02:43: prep: echo ":d:"
			:d:
			>> done (118.9849ms)
			17:02:43: prep: echo ":e:"
			:e:
			>> done (119.2557ms)
			*/
			ret := events(lt.String())
			if reflect.DeepEqual(ret, expected) {
				break
			}
			if time.Now().Sub(start) > timeout {
				t.Errorf("Expected\n%#v\nGot\n%#v", expected, ret)
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		modchan <- nil
	}

	mr := ModRunner{
		Log:    lt.Log,
		Config: cnf,
	}

	err = mr.runOnChan(modchan, cback)
	if err != nil {
		t.Fatalf("runOnChan: %s", err)
	}
	ret := events(lt.String())
	if !reflect.DeepEqual(ret, expected) {
		t.Errorf("Expected\n%#v\nGot\n%#v", expected, ret)
	}
}

func TestWatch(t *testing.T) {
	t.Run(
		"single",
		func(t *testing.T) {
			_testWatch(
				t,
				func() { touch("a/touched") },
				[]string{
					":all: ./a/initial",
					":a: ./a/initial",
					":skipit: ./a/touched",
					":all: ./a/touched",
					":a: ./a/touched",
				},
			)
		},
	)
	//t.Run(
	//	"double",
	//	func(t *testing.T) {
	//		_testWatch(
	//			t,
	//			func() {
	//				touch("a/touched")
	//				touch("b/touched")
	//			},
	//			[]string{
	//				":all: ./a/initial",
	//				":a: ./a/initial",
	//				":skipit: ./a/touched ./b/touched",
	//				":all: ./a/touched ./b/touched",
	//				":a: ./a/touched",
	//				":b: ./b/touched",
	//			},
	//		)
	//	},
	//)
	//t.Run(
	//	"inner",
	//	func(t *testing.T) {
	//		_testWatch(
	//			t,
	//			func() {
	//				touch("a/inner/touched.xxx")
	//			},
	//			[]string{
	//				":all: ./a/initial",
	//				":a: ./a/initial",
	//				":skipit: ./a/inner/touched.xxx",
	//				":all: ./a/inner/touched.xxx",
	//				":c: ./a/inner/touched.xxx",
	//			},
	//		)
	//	},
	//)
	//t.Run(
	//	"direct",
	//	func(t *testing.T) {
	//		_testWatch(
	//			t,
	//			func() {
	//				touch("a/direct")
	//			},
	//			[]string{
	//				":all: ./a/initial",
	//				":a: ./a/initial",
	//				":skipit: ./a/direct",
	//				":all: ./a/direct",
	//				":a: ./a/direct",
	//				":d: ./a/direct",
	//			},
	//		)
	//	},
	//)
	//t.Run(
	//	"rootdirect",
	//	func(t *testing.T) {
	//		_testWatch(
	//			t,
	//			func() {
	//				touch("direct")
	//			},
	//			[]string{
	//				":all: ./a/initial",
	//				":a: ./a/initial",
	//				":skipit: ./direct",
	//				":all: ./direct",
	//				":e: ./direct",
	//			},
	//		)
	//	},
	//)
}
