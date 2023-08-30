package modd

import (
	"time"

	"github.com/cortesi/modd/conf"
	"github.com/cortesi/modd/notify"
	"github.com/cortesi/modd/shell"
	"github.com/cortesi/modd/varcmd"
	"github.com/cortesi/moddwatch"
	"github.com/cortesi/termlog"
)

// ProcError is a process error, possibly containing command output
type ProcError struct {
	shorttext string
	Output    string
}

func (p ProcError) Error() string {
	return p.shorttext
}

// RunProc runs a process to completion, sending output to log
// 运行进程，直到结束
func RunProc(cmd string, shellMethod string, dir string, log termlog.Stream) error {
	log.Header()
	ex, err := shell.NewExecutor(shellMethod, cmd, dir)
	if err != nil {
		return err
	}
	start := time.Now()
	err, estate := ex.Run(log, true)
	if err != nil {
		return err
	} else if estate.Error != nil {
		log.Shout("%s", estate.Error)
		return ProcError{estate.Error.Error(), estate.ErrOutput}
	}
	log.Notice(">> done11 (%s)", time.Since(start))
	return nil
}

// RunPreps runs all commands in sequence. Stops if any command returns an error.
// 按顺序运行所有命令。如果任何命令返回错误，则停止
func RunPreps(
	b conf.Block,
	vars map[string]string,
	mod *moddwatch.Mod,
	log termlog.TermLog,
	notifiers []notify.Notifier,
	initial bool,
) error {
	//验证shell变量名称是否合规
	sh, err := shell.GetShellName(vars[shellVarName])
	if err != nil {
		return err
	}

	//获取所有修改（编辑|添加）文件切片，不包括删除
	var modified []string
	if mod != nil {
		modified = mod.All()
	}

	// 获取block内的一组变量
	vcmd := varcmd.VarCmd{Block: &b, Modified: modified, Vars: vars}
	for _, p := range b.Preps {
		cmd, err := vcmd.Render(p.Command)
		//如果配置Onchange,则首次运行不做操作，后续操作输出跳过日志
		if initial && p.Onchange {
			log.Say(niceHeader("skipping prep: ", cmd))
			continue
		}
		if err != nil {
			return err
		}
		err = RunProc(cmd, sh, b.InDir, log.Stream(niceHeader("prep: ", cmd)))
		if err != nil {
			if pe, ok := err.(ProcError); ok {
				for _, n := range notifiers {
					n.Push("modd error", pe.Output, "")
				}
			}
			return err
		}
	}
	return nil
}
