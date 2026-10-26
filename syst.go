package main

import (
	"fmt"
	"html"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	tele "gopkg.in/telebot.v3"
)

var (
	sysValidEmojiOnce sync.Once
	sysValidEmojiIDs  map[string]bool
	sysBotInfoOnce    sync.Once
	sysBotInfo        map[string]interface{}
)

const (
	sysHeaderEmojiID = "5445224894386172410"
)

var sysEmojiMap = map[string]string{
	"header": sysHeaderEmojiID,
}

func sysLoadValidEmojiIDs(bot *tele.Bot) map[string]bool {
	sysValidEmojiOnce.Do(func() {
		sysValidEmojiIDs = map[string]bool{}
		ids := make([]string, 0, len(sysEmojiMap))
		seen := map[string]bool{}
		for _, v := range sysEmojiMap {
			if v == "" {
				continue
			}
			if !seen[v] {
				seen[v] = true
				ids = append(ids, v)
			}
		}
		if len(ids) == 0 {
			return
		}
		// Try to validate via Telegram API if available
		// telebot.v3 may not expose GetCustomEmojiStickers in all versions; best effort
		type customEmojiGetter interface {
			GetCustomEmojiStickers([]string) ([]tele.Sticker, error)
		}
		if g, ok := interface{}(bot).(customEmojiGetter); ok {
			for i := 0; i < len(ids); i += 200 {
				end := i + 200
				if end > len(ids) {
					end = len(ids)
				}
				stks, err := g.GetCustomEmojiStickers(ids[i:end])
				if err != nil {
					continue
				}
				for _, s := range stks {
					if s != nil && s.CustomEmojiID != "" {
						sysValidEmojiIDs[s.CustomEmojiID] = true
					}
				}
			}
			return
		}
		// Fallback: assume valid to keep identical tg-emoji tags
		for _, id := range ids {
			sysValidEmojiIDs[id] = true
		}
	})
	if sysValidEmojiIDs == nil {
		sysValidEmojiIDs = map[string]bool{}
	}
	return sysValidEmojiIDs
}

func sysTg(emojiID string, fallback string) string {
	valid := sysLoadValidEmojiIDs(nil)
	if valid != nil && emojiID != "" {
		if ok := valid[emojiID]; ok {
			return `<tg-emoji emoji-id="` + html.EscapeString(emojiID) + `">` + fallback + `</tg-emoji>`
		}
		return fallback
	}
	if emojiID != "" {
		return `<tg-emoji emoji-id="` + html.EscapeString(emojiID) + `">` + fallback + `</tg-emoji>`
	}
	return fallback
}

func sysEsc(v interface{}) string {
	switch t := v.(type) {
	case string:
		return html.EscapeString(t)
	case int:
		return html.EscapeString(strconv.Itoa(t))
	case int64:
		return html.EscapeString(strconv.FormatInt(t, 10))
	case uint64:
		return html.EscapeString(strconv.FormatUint(t, 10))
	case float64:
		return html.EscapeString(strconv.FormatFloat(t, 'f', -1, 64))
	default:
		return html.EscapeString(fmt.Sprintf("%v", t))
	}
}

func sysOSInfo() map[string]string {
	hostname, _ := os.Hostname()
	return map[string]string{
		"system":   runtime.GOOS,
		"release":  "",
		"version":  "",
		"machine":  runtime.GOARCH,
		"hostname": hostname,
	}
}

func sysGoInfo() map[string]string {
	return map[string]string{
		"version": runtime.Version(),
	}
}

func sysCPUInfo() map[string]interface{} {
	info := map[string]interface{}{
		"cores_logical":  runtime.NumCPU(),
		"cores_physical": runtime.NumCPU(),
		"usage_percent":  interface{}(nil),
		"freq_mhz":       interface{}(nil),
		"model":          "Unknown",
	}
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			low := strings.ToLower(line)
			if strings.HasPrefix(low, "model name") || strings.HasPrefix(low, "hardware") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					info["model"] = strings.TrimSpace(parts[1])
					break
				}
			}
		}
	}
	return info
}

func sysRAMInfo() map[string]float64 {
	info := map[string]float64{"total_mb": 0, "used_mb": 0, "percent": 0}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// total not directly available, use sysinfo? best effort
	info["used_mb"] = float64(m.Alloc) / 1024.0 / 1024.0
	return info
}

func sysDiskInfo(path string) map[string]float64 {
	info := map[string]float64{"total_gb": 0, "used_gb": 0, "free_gb": 0, "percent": 0}
	return info
}

func sysProcInfo() map[string]interface{} {
	info := map[string]interface{}{
		"pid":         os.Getpid(),
		"rss_mb":      0.0,
		"cpu_percent": 0,
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	info["rss_mb"] = float64(m.Sys) / 1024.0 / 1024.0
	return info
}

func sysLoadAverage() map[string]float64 {
	if runtime.GOOS != "linux" {
		return nil
	}
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return nil
	}
	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return nil
	}
	l1, e1 := strconv.ParseFloat(fields[0], 64)
	l5, e2 := strconv.ParseFloat(fields[1], 64)
	l15, e3 := strconv.ParseFloat(fields[2], 64)
	if e1 != nil || e2 != nil || e3 != nil {
		return nil
	}
	return map[string]float64{"1m": l1, "5m": l5, "15m": l15}
}

func sysNetInfo() map[string]uint64 {
	info := map[string]uint64{"bytes_sent": 0, "bytes_recv": 0}
	return info
}

func sysBotInfo(bot *tele.Bot) map[string]interface{} {
	sysBotInfoOnce.Do(func() {
		me, err := bot.Me()
		if err != nil || me == nil {
			sysBotInfo = map[string]interface{}{
				"username": "@dazai",
				"id":       0,
				"is_premium": false,
				"lib":      "gopkg.in/telebot.v3",
				"lib_version": "v3",
			}
			return
		}
		uname := me.Username
		if uname != "" {
			uname = "@" + uname
		} else if me.FirstName != "" {
			uname = me.FirstName
		}
		sysBotInfo = map[string]interface{}{
			"username":     uname,
			"id":           me.ID,
			"is_premium":   false,
			"lib":          "gopkg.in/telebot.v3",
			"lib_version":  "v3",
		}
	})
	if sysBotInfo == nil {
		sysBotInfo = map[string]interface{}{
			"username":     "@dazai",
			"id":           0,
			"is_premium":   false,
			"lib":          "gopkg.in/telebot.v3",
			"lib_version":  "v3",
		}
	}
	return sysBotInfo
}

func sysSendSafe(bot *tele.Bot, m *tele.Message, text string) {
	// Try normal send via _reply path
	_, err := _reply(bot, m, text)
	if err == nil {
		return
	}
	// Fallback: strip tg-emoji
	// simple regex-like replacement
	for {
		start := strings.Index(text, `<tg-emoji`)
		if start == -1 {
			break
		}
		endTag := strings.Index(text[start:], `>`)
		if endTag == -1 {
			break
		}
		endTag += start
		close := strings.Index(text[endTag:], `</tg-emoji>`)
		if close == -1 {
			break
		}
		closeIdx := endTag + close + len(`</tg-emoji>`)
		innerStart := strings.Index(text[endTag:closeIdx], `>`)
		if innerStart == -1 {
			break
		}
		inner := text[endTag+innerStart+1 : endTag+close]
		text = text[:start] + inner + text[closeIdx:]
	}
	_reply(bot, m, text)
}

func SetupSys(bot *tele.Bot) {
	sysLoadValidEmojiIDs(bot)
	sysBotInfo(bot)

	bot.Handle("/sys", func(c tele.Context) error {
		m := c.Message()
		if m == nil {
			return nil
		}
		return sysRun(bot, m)
	})
	bot.Handle("/system", func(c tele.Context) error {
		m := c.Message()
		if m == nil {
			return nil
		}
		return sysRun(bot, m)
	})
	bot.Handle("/status", func(c tele.Context) error {
		m := c.Message()
		if m == nil {
			return nil
		}
		return sysRun(bot, m)
	})
}

func sysRun(bot *tele.Bot, message *tele.Message) error {
	osI := sysOSInfo()
	goI := sysGoInfo()
	cpuI := sysCPUInfo()
	ramI := sysRAMInfo()
	diskI := sysDiskInfo("/")
	procI := sysProcInfo()
	loadI := sysLoadAverage()
	netI := sysNetInfo()
	botI := sysBotInfo(bot)

	cpuUsage := "N/A"
	if v, ok := cpuI["usage_percent"]; ok && v != nil {
		cpuUsage = fmt.Sprintf("%v%%", v)
	}

	lines := []string{
		"<b>" + sysTg(sysEmojiMap["header"], "🖥") + " System Information</b>",
		"━━━━━━━━━━━━━━━━━━",
		"<b>OS:</b> <b><code>" + sysEsc(osI["system"]) + " " + sysEsc(osI["release"]) + "</code></b>",
		"<b>Hostname:</b> <b><code>" + sysEsc(osI["hostname"]) + "</code></b>",
		"<b>Arch:</b> <b><code>" + sysEsc(osI["machine"]) + "</code></b>",
		"<b>Go:</b> <b><code>" + sysEsc(goI["version"]) + "</code></b>",
		"<b>Lib:</b> <b><code>" + sysEsc(botI["lib"]) + " " + sysEsc(botI["lib_version"]) + "</code></b>",
		"━━━━━━━━━━━━━━━━━━",
		"<b>CPU:</b> <b><code>" + sysEsc(cpuI["model"]) + "</code></b>",
		"<b>CPU Usage:</b> <b><code>" + sysEsc(cpuUsage) + "</code></b>",
		"<b>Cores:</b> <b><code>" + fmt.Sprintf("%vP / %vL", cpuI["cores_physical"], cpuI["cores_logical"]) + "</code></b>",
	}

	if v, ok := cpuI["freq_mhz"]; ok && v != nil {
		lines = append(lines, "<b>CPU Freq:</b> <b><code>"+sysEsc(v)+" MHz</code></b>")
	}

	lines = append(lines,
		"<b>RAM:</b> <b><code>"+fmt.Sprintf("%.1f", ramI["used_mb"])+" / "+fmt.Sprintf("%.1f", ramI["total_mb"])+" MB ("+fmt.Sprintf("%.0f", ramI["percent"])+"%)</code></b>",
		"<b>Disk:</b> <b><code>"+fmt.Sprintf("%.2f", diskI["used_gb"])+" / "+fmt.Sprintf("%.2f", diskI["total_gb"])+" GB ("+fmt.Sprintf("%.0f", diskI["percent"])+"%)</code></b>",
	)

	if loadI != nil {
		lines = append(lines, "<b>Load Avg:</b> <b><code>"+fmt.Sprintf("%.2f", loadI["1m"])+" / "+fmt.Sprintf("%.2f", loadI["5m"])+" / "+fmt.Sprintf("%.2f", loadI["15m"])+"</code></b>")
	}

	lines = append(lines,
		"━━━━━━━━━━━━━━━━━━",
		"<b>Bot:</b> <b><code>"+sysEsc(botI["username"])+"</code></b>",
		"<b>Bot ID:</b> <b><code>"+fmt.Sprintf("%v", botI["id"])+"</code></b>",
		"<b>Premium:</b> <b><code>"+func() string {
			if b, ok := botI["is_premium"].(bool); ok && b {
				return "Yes"
			}
			return "No"
		}()+"</code></b>",
		"<b>PID:</b> <b><code>"+fmt.Sprintf("%v", procI["pid"])+"</code></b>",
		"<b>RSS:</b> <b><code>"+fmt.Sprintf("%.1f", procI["rss_mb"].(float64))+" MB</code></b>",
		"<b>Net Sent:</b> <b><code>"+fmt.Sprintf("%d", netI["bytes_sent"])+", B</code></b>",
		"<b>Net Recv:</b> <b><code>"+fmt.Sprintf("%d", netI["bytes_recv"])+", B</code></b>",
		"<b>Server Time:</b> <b><code>"+time.Now().Format("2006-01-02 15:04:05")+"</code></b>",
		"━━━━━━━━━━━━━━━━━━",
	)

	sysSendSafe(bot, message, strings.Join(lines, "\n"))
	return nil
}
