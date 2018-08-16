package humanlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

// JSONHandler can handle logs emmited by logrus.TextFormatter loggers.
type JSONHandler struct {
	buf     *bytes.Buffer
	out     *tabwriter.Writer
	truncKV int

	Opts *HandlerOptions

	Level   string
	Time    time.Time
	Message string
	Fields  map[string]string

	last map[string]string
}

func (h *JSONHandler) clear() {
	h.Level = ""
	h.Time = time.Time{}
	h.Message = ""
	h.last = h.Fields
	h.Fields = make(map[string]string)
	if h.buf != nil {
		h.buf.Reset()
	}
}

// TryHandle tells if this line was handled by this handler.
func (h *JSONHandler) TryHandle(d []byte) bool {
	if !bytes.Contains(d, []byte(`"time":`)) && !bytes.Contains(d, []byte(`"ts":`)) && !bytes.Contains(d, []byte(`"timestamp":`)) {
		return false
	}
	err := h.UnmarshalJSON(d)
	if err != nil {
		h.clear()
		return false
	}
	return true
}

// UnmarshalJSON sets the fields of the handler.
func (h *JSONHandler) UnmarshalJSON(data []byte) error {
	raw := make(map[string]interface{})
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	time, ok := raw["time"]
	if ok {
		delete(raw, "time")
	} else {
		time, ok = raw["ts"]
		if ok {
			delete(raw, "ts")
		} else {
			time, ok = raw["timestamp"]
			if ok {
				delete(raw, "timestamp")
			}
		}
	}
	if ok {
		h.Time, ok = tryParseTime(time)
		if !ok {
			return fmt.Errorf("field time is not a known timestamp: %v", time)
		}
	}
	if h.Message, ok = raw["msg"].(string); ok {
		delete(raw, "msg")
	} else if h.Message, ok = raw["message"].(string); ok {
		delete(raw, "message")
	}

	h.Level, ok = raw["level"].(string)
	if !ok {
		h.Level, ok = raw["lvl"].(string)
		delete(raw, "lvl")
		if !ok {
			h.Level = "???"
		}
	} else {
		delete(raw, "level")
	}

	if h.Fields == nil {
		h.Fields = make(map[string]string)
	}

	for key, val := range raw {
		switch v := val.(type) {
		case float64:
			if v-math.Floor(v) < 0.000001 && v < 1e9 {
				// looks like an integer that's not too large
				h.Fields[key] = fmt.Sprintf("%d", int(v))
			} else {
				h.Fields[key] = fmt.Sprintf("%g", v)
			}
		case string:
			h.Fields[key] = fmt.Sprintf("%q", v)
		default:
			h.Fields[key] = fmt.Sprintf("%v", v)
		}
	}

	return nil
}

// Prettify the output in a logrus like fashion.
func (h *JSONHandler) Prettify(skipUnchanged bool) []byte {
	defer h.clear()
	if h.out == nil {
		if h.Opts == nil {
			h.Opts = DefaultOptions
		}
		h.buf = bytes.NewBuffer(nil)
		h.out = tabwriter.NewWriter(h.buf, 0, 1, 0, '\t', 0)
	}

	var msg string
	if h.Message == "" {
		msg = h.Opts.MsgAbsentColor.Sprint("<no msg>")
	} else {
		msg = h.Opts.MsgColor.Sprint(h.Message)
	}

	lvl := strings.ToUpper(h.Level)[:imin(4, len(h.Level))]
	var level string
	switch h.Level {
	case "debug":
		level = h.Opts.DebugLevelColor.Sprint(lvl)
	case "info":
		level = h.Opts.InfoLevelColor.Sprint(lvl)
	case "warn", "warning":
		level = h.Opts.WarnLevelColor.Sprint(lvl)
	case "error":
		level = h.Opts.ErrorLevelColor.Sprint(lvl)
	case "fatal", "panic":
		level = h.Opts.FatalLevelColor.Sprint(lvl)
	default:
		level = h.Opts.UnknownLevelColor.Sprint(lvl)
	}

	_, _ = fmt.Fprintf(h.out, "%s |%s| %s\t %s",
		h.Opts.TimeColor.Sprint(h.Time.Format(h.Opts.TimeFormat)),
		level,
		msg,
		strings.Join(h.joinKVs(skipUnchanged, "="), "\t "),
	)

	_ = h.out.Flush()

	return h.buf.Bytes()
}

func (h *JSONHandler) joinKVs(skipUnchanged bool, sep string) []string {

	kv := make([]string, 0, len(h.Fields))
	for k, v := range h.Fields {
		if !h.Opts.shouldShowKey(k) {
			continue
		}

		if skipUnchanged {
			if lastV, ok := h.last[k]; ok && lastV == v && !h.Opts.shouldShowUnchanged(k) {
				continue
			}
		}
		kstr := h.Opts.KeyColor.Sprint(k)

		var vstr string
		if h.Opts.Truncates && len(v) > h.Opts.TruncateLength {
			vstr = v[:h.Opts.TruncateLength] + "..."
		} else {
			vstr = v
		}
		vstr = h.Opts.ValColor.Sprint(vstr)
		kv = append(kv, kstr+sep+vstr)
	}

	sort.Strings(kv)

	if h.Opts.SortLongest {
		sort.Stable(byLongest(kv))
	}

	return kv
}
