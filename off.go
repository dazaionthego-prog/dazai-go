package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	tele "gopkg.in/telebot.v3"
)

var (
	offMu       sync.RWMutex
	disabledSet = map[string]bool{}
	offFilePath string
)

const (
	maintHTML = "<tg-emoji emoji-id=\"5440660757194744323\">‼️</tg-emoji><b> Under Maintenance!</b>\n━━━━━━━━━━━━━━━━━━\n<b>This command has been turned off by the admins. Please try again later.</b>\n━━━━━━━━━━━━━━━━━━"
)

var offProtected = map[string]bool{
	"on":    true,
	"off":   true,
	"start": true,
}

func offInit() {
	if offFilePath == "" {
		exe, err := os.Executable()
		if err != nil {
			wd, _ := os.Getwd()
			offFilePath = filepath.Join(wd, "disabled.json")
		} else {
			dir := filepath.Dir(exe)
			if dir == "." {
				wd, _ := os.Getwd()
				dir = wd
			}
			offFilePath = filepath.Join(dir, "disabled.json")
		}
	}
	data, err := os.ReadFile(offFilePath)
	if err != nil {
		disabledSet = map[string]bool{}
		return
	}
	var arr []string
	if json.Unmarshal(data, &arr) != nil {
		disabledSet = map[string]bool{}
		return
	}
	disabledSet = map[string]bool{}
	for _, v := range arr {
		v = strings.ToLower(strings.TrimSpace(v))
		if v != "" {
			disabledSet[v] = true
		}
	}
}

func offSave() {
	offMu.RLock()
	arr := make([]string, 0, len(disabledSet))
	for k, v := range disabledSet {
		if v {
			arr = append(arr, k)
		}
	}
	offMu.RUnlock()
	sort.Strings(arr)
	b, _ := json.Marshal(arr)
	tmp := offFilePath + ".tmp"
	_ = os.WriteFile(tmp, b, 0644)
	_ = os.Rename(tmp, offFilePath)
}

func offCmdName(m *tele.Message) string {
	if m == nil || m.Text == "" {
		return ""
	}
	text := m.Text
	if !strings.HasPrefix(text, "/") {
		return ""
	}
	body := strings.Fields(text)
	if len(body) == 0 {
		return ""
	}
	cmd := body[0][1:]
	if idx := strings.Index(cmd, "@"); idx != -1 {
		cmd = cmd[:idx]
	}
	return strings.ToLower(cmd)
}

func offIsBlocked(m *tele.Message) bool {
	name := offCmdName(m)
	if name == "" {
		return false
	}
	if m.Sender != nil && _isAdmin(m) {
		return false
	}
	offMu.RLock()
	defer offMu.RUnlock()
	return disabledSet[name]
}

func SetupOff(bot *tele.Bot) {
	offInit()

	bot.Handle("/off", func(c tele.Context) error {
		m := c.Message()
		if m == nil {
			return nil
		}
		if !_isAdmin(m) {
			return nil
		}
		args := strings.Fields(m.Text)
		if len(args) == 1 {
			offMu.RLock()
			items := make([]string, 0)
			for k, v := range disabledSet {
				if v {
					items = append(items, k)
				}
			}
			offMu.RUnlock()
			sort.Strings(items)
			if len(items) == 0 {
				_reply(bot, m, "<b>No commands are disabled.</b>")
				return nil
			}
			lines := make([]string, 0, len(items))
			for _, it := range items {
				lines = append(lines, "• <code>/"+it+"</code>")
			}
			_reply(bot, m, "<b>Disabled commands:</b>\n"+strings.Join(lines, "\n"))
			return nil
		}
		name := strings.TrimPrefix(args[1], "/")
		name = strings.ToLower(name)
		if offProtected[name] {
			_reply(bot, m, "<b>/"+name+" can't be disabled.</b>")
			return nil
		}
		offMu.Lock()
		disabledSet[name] = true
		offMu.Unlock()
		offSave()
		_reply(bot, m, "🔴 <b>/"+name+" has been turned off.</b>")
		return nil
	})

	bot.Handle("/on", func(c tele.Context) error {
		m := c.Message()
		if m == nil {
			return nil
		}
		if !_isAdmin(m) {
			return nil
		}
		args := strings.Fields(m.Text)
		if len(args) == 1 {
			_reply(bot, m, "Usage: <code>/on {command}</code>")
			return nil
		}
		name := strings.TrimPrefix(args[1], "/")
		name = strings.ToLower(name)
		offMu.Lock()
		if !disabledSet[name] {
			offMu.Unlock()
			_reply(bot, m, "<b>/"+name+" is already on.</b>")
			return nil
		}
		delete(disabledSet, name)
		offMu.Unlock()
		offSave()
		_reply(bot, m, "🟢 <b>/"+name+" has been turned back on.</b>")
		return nil
	})

	bot.Use(func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			m := c.Message()
			if m != nil && strings.HasPrefix(m.Text, "/") {
				if offIsBlocked(m) {
					_reply(bot, m, maintHTML)
					return nil
				}
			}
			return next(c)
		}
	})
}
