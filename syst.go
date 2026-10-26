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
	if sy
