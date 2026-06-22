package myn

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func RepeatLabel(rule string) string {
	if strings.TrimSpace(rule) == "" {
		return "-"
	}
	if strings.EqualFold(rule, "PARENT") {
		return "with parent"
	}

	parts := parseRepeatRule(rule)
	freq := parts["FREQ"]
	if freq == "" {
		return rule
	}

	interval := 1
	if value := parts["INTERVAL"]; value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			interval = parsed
		}
	}

	label := frequencyLabel(freq, interval)
	if label == "" {
		return rule
	}

	if byDay := byDayLabel(parts["BYDAY"]); byDay != "" {
		label += " on " + byDay
	}
	if byMonthDay := parts["BYMONTHDAY"]; byMonthDay != "" {
		label += " on day " + byMonthDay
	}
	if byMonth := byMonthLabel(parts["BYMONTH"]); byMonth != "" {
		label += " in " + byMonth
	}
	if count := parts["COUNT"]; count != "" {
		label += " for " + count + " times"
	}
	if until := parts["UNTIL"]; until != "" {
		label += " until " + until
	}
	if _, ok := parts["FROMCOMP"]; ok {
		label += " from completion"
	}
	if _, ok := parts["FASTFORWARD"]; ok {
		label += " fast-forward"
	}

	return label
}

func parseRepeatRule(rule string) map[string]string {
	parts := make(map[string]string)
	for _, part := range strings.Split(rule, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, ok := strings.Cut(part, "=")
		key = strings.ToUpper(strings.TrimSpace(key))
		if !ok {
			parts[key] = ""
			continue
		}
		parts[key] = strings.ToUpper(strings.TrimSpace(value))
	}
	return parts
}

func frequencyLabel(freq string, interval int) string {
	labels := map[string]struct {
		adverb string
		unit   string
	}{
		"DAILY":    {adverb: "daily", unit: "day"},
		"WEEKLY":   {adverb: "weekly", unit: "week"},
		"MONTHLY":  {adverb: "monthly", unit: "month"},
		"YEARLY":   {adverb: "yearly", unit: "year"},
		"HOURLY":   {adverb: "hourly", unit: "hour"},
		"MINUTELY": {adverb: "every minute", unit: "minute"},
	}
	label, ok := labels[freq]
	if !ok {
		return ""
	}
	if interval == 1 {
		return label.adverb
	}
	return fmt.Sprintf("every %d %ss", interval, label.unit)
}

func byDayLabel(value string) string {
	if value == "" {
		return ""
	}
	names := map[string]string{
		"MO": "Mon",
		"TU": "Tue",
		"WE": "Wed",
		"TH": "Thu",
		"FR": "Fri",
		"SA": "Sat",
		"SU": "Sun",
	}
	days := strings.Split(value, ",")
	labels := make([]string, 0, len(days))
	for _, day := range days {
		day = strings.TrimSpace(day)
		prefix := strings.TrimRight(day, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
		code := strings.TrimLeft(day, "+-0123456789")
		name := names[code]
		if name == "" {
			name = day
		}
		if prefix != "" && name != day {
			name = ordinal(prefix) + " " + name
		}
		labels = append(labels, name)
	}
	return strings.Join(labels, ", ")
}

func byMonthLabel(value string) string {
	if value == "" {
		return ""
	}
	names := map[string]string{
		"1": "Jan", "2": "Feb", "3": "Mar", "4": "Apr", "5": "May", "6": "Jun",
		"7": "Jul", "8": "Aug", "9": "Sep", "10": "Oct", "11": "Nov", "12": "Dec",
	}
	months := strings.Split(value, ",")
	labels := make([]string, 0, len(months))
	for _, month := range months {
		month = strings.TrimSpace(month)
		if name := names[month]; name != "" {
			labels = append(labels, name)
		} else {
			labels = append(labels, month)
		}
	}
	return strings.Join(labels, ", ")
}

func ordinal(value string) string {
	n, err := strconv.Atoi(value)
	if err != nil {
		return value
	}
	if n < 0 {
		return "last"
	}
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suffix)
}

func KnownRepeatKeys(rule string) []string {
	parts := parseRepeatRule(rule)
	keys := make([]string, 0, len(parts))
	for key := range parts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
