package putils

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/EICHI-X/ptools/logs"
	"github.com/bytedance/sonic"
)

// 带时区
const TimeFormatZone = "2006-01-02T15:04:05Z07:00"

// 微秒，千分之一毫秒
const TimeFormatMicrosecond = "2006-01-02 15:04:05.000000"
const TimeFormatDateF = "2006-01-02"
const TimeFormatDate = "20060102"
const TimeFormat = "2006-01-02 15:04:05"

var TimeFormatMap = map[string]string{
	"2006-01-02 15:04:05": "2006-01-02 15:04:05",
	"2006-01-02":          "2006-01-02",
	"20060102":            "20060102",
}

// ToMap 结构体转为Map[string]interface{}, tagName 表示tag字段，比如json，gorm
func ToMap(in interface{}, tagName string, filterKey map[string]bool, out map[string]interface{}) (map[string]interface{}, error) {
	if out == nil {
		out = make(map[string]interface{})
	}

	v := reflect.ValueOf(in)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct { // 非结构体返回错误提示
		return nil, fmt.Errorf("ToMap only accepts struct or struct pointer; got %T", v)
	}

	t := v.Type()
	// 遍历结构体字段
	// 指定tagName值为map中key;字段值为map中value
	for i := 0; i < v.NumField(); i++ {
		fi := t.Field(i)
		if fi.Anonymous {
			out, _ = ToMap(v.Field(i).Interface(), tagName, filterKey, out)
		}
		if tagValue := fi.Tag.Get(tagName); tagValue != "" {
			if len(filterKey) > 0 {
				if _, ok := filterKey[tagValue]; ok {
					continue
				}
			}
			out[tagValue] = v.Field(i).Interface()
		}
	}
	return out, nil
}
func ToJson(v interface{}) string {
	if v == nil {
		return ""

	}
	r, _ := json.Marshal(v)
	return string(r)
}
func ToJsonSonic(v interface{}) string {
	if v == nil {
		return ""
	}
	r, _ := sonic.MarshalString(v)
	return r
}

func TimeCost(ctx context.Context) func() {
	pc := make([]uintptr, 1)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	name := f.Name()
	start := time.Now()
	return func() {
		tc := time.Since(start)
		logs.CtxInfof(ctx, "%v  timecost S = %vs ,Ms = %v ms", name, tc.Seconds(), tc.Milliseconds())
	}
}
func TimeCostWithMsg(ctx context.Context, msg string) func() {
	pc := make([]uintptr, 1)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	name := f.Name()
	start := time.Now()
	return func() {
		tc := time.Since(start)
		logs.CtxInfof(ctx, "%v  timecost S = %vs ,Ms = %v ms msg:%v", name, tc.Seconds(), tc.Milliseconds(), msg)
	}
}
func MaxInt(a, b int) int {
	if a >= b {
		return a
	}
	return b
}
func MinInt(a, b int) int {
	if a <= b {
		return a
	}
	return b
}
func MaxFloat(a, b float64) float64 {
	if a >= b {
		return a
	}
	return b
}

func DecimalPct(value float64) float64 {
	value, _ = strconv.ParseFloat(fmt.Sprintf("%.4f", value*100), 64)
	return value
}
func DecimalFloatToStr(value interface{}, offset int) string {
	if v, ok := value.(float64); ok {
		a := "%." + fmt.Sprint(offset) + "f"
		r := fmt.Sprintf(a, v)
		return r
	} else {
		return fmt.Sprint(value)
	}

}

// DecimalInterface 浮点数精度设置为小数点后2位
func DecimalInterface(value interface{}) (interface{}, error) {
	if v, ok := value.(float64); ok {
		v, err := strconv.ParseFloat(fmt.Sprintf("%.2f", v), 64)
		return v, err
	} else {
		err := fmt.Errorf(" %v is not float ", value)
		return value, err
	}

}
func DecimalMap(data []map[string]interface{}, columns []string) []map[string]interface{} {
	if len(data) == 0 {
		return data
	}
	for _, c := range columns {
		if _, ok := data[0][c]; !ok {
			continue
		}
		for idx, fields := range data {
			if v, ok := fields[c]; ok {
				if t, err := DecimalInterface(v); err == nil {
					data[idx][c] = t
				}
			}

		}
	}

	return data

}
func FormatTimeToDate(t string, format string) string {
	if len(t) > len("2022-02-02") {
		tranTime, _ := time.Parse(TimeFormat, t)
		t = tranTime.Format(format)
	} else if len(t) > len("20060102") {
		tranTime, _ := time.Parse(TimeFormatDateF, t)
		t = tranTime.Format(format)
	} else if len(TimeFormatDate) == len(t) {
		tranTime, _ := time.Parse(TimeFormatDate, t)
		t = tranTime.Format(format)
	}
	return t
}
func DateTime(now time.Time) int64 {
	d := now.Format(TimeFormatDate)
	r, _ := strconv.ParseInt(d, 10, 64)
	return r
}

// "2006-01-02 15:04:05.000000" 微秒级别,毫秒千分之一
func FormatTimeMicrosecond(now time.Time) string {
	d := now.Format(TimeFormatMicrosecond)

	return d
}
func ParseTime(t string) time.Time {
	if len(t) == len(TimeFormatZone) {
		tranTime, _ := time.Parse(TimeFormatZone, t)
		return tranTime
	} else if len(t) == len(TimeFormatMicrosecond) {
		tranTime, _ := time.Parse(TimeFormatMicrosecond, t)
		return tranTime
	} else if len(t) == len(TimeFormat) {
		tranTime, _ := time.Parse(TimeFormat, t)
		return tranTime
	} else if len(t) == len(TimeFormatDateF) {
		tranTime, _ := time.Parse(TimeFormatDateF, t)
		return tranTime
	} else if len(t) == len(TimeFormatDate) {
		tranTime, _ := time.Parse(TimeFormatDate, t)
		return tranTime
	} else if len(t) == len(TimeFormatZone) {
		tranTime, _ := time.Parse(TimeFormatZone, t)
		return tranTime
	} else if len(t) == len(time.RFC3339) {
		tranTime, _ := time.Parse(time.RFC3339, t)
		return tranTime
	} else if len(t) == len(time.RFC1123) {
		tranTime, _ := time.Parse(time.RFC1123, t)
		return tranTime
	} else if len(t) == len(time.RFC1123Z) {
		tranTime, _ := time.Parse(time.RFC1123Z, t)
		return tranTime
	} else if len(t) == len(time.RFC850) {
		tranTime, _ := time.Parse(time.RFC850, t)
		return tranTime
	} else {
		formats := []string{
			// Monday, 02-Jan-06 15:04:05 MST
			"2006-01-02 15:04:05",                 // 2025-01-02 01:44:56
			"2006-01-02",                          // 2025-01-02
			"02 Jan 2006 15:04:05",                // 02 Jan 2025 01:44:56
			"02 Jan 2006",                         // 02 Jan 2025
			"02/01/2006",                          // 02/01/2025
			"01-02-2006",                          // 01-02-2025
			"2006/01/02",                          // 2025/01/02
			"2006.01.02",                          // 2025.01.02
			"2006-01-02 15:04:05.999999999",       // 2025-01-02 01:44:56.853
			"2006-01-02 15:04:05.999999999Z07:00", // 2025-01-02 01:44:56.853Z
			"2006-01-02T15:04:05Z07:00",           // 2025-01-02T01:44:56Z
		}

		for _, format := range formats {
			parsedTime, err := time.Parse(format, t)
			if err == nil {
				return parsedTime // 成功解析，返回结果
			}
		}
		return time.Time{}
	}
}
func PtrStr(v string) *string {
	return &v
}
func PtrInt(v int) *int {
	return &v
}
func PtrInt64(v int64) *int64 {
	return &v
}
func PtrInt16(v int16) *int16 {
	return &v
}
func PtrInt32(v int32) *int32 {
	return &v
}
func PtrFloat(v float64) *float64 {
	return &v
}
func PtrBool(v bool) *bool {
	return &v
}
func Int64ToStrSlice(v []int64) []string {
	r := make([]string, len(v))
	for idx, val := range v {
		r[idx] = strconv.FormatInt(val, 10)
	}
	return r
}
func IntToStrSlice(v []int) []string {
	r := make([]string, len(v))
	for idx, val := range v {
		r[idx] = strconv.Itoa(val)
	}
	return r
}
func StrToInt64Slice(v []string) ([]int64, []error) {
	r := make([]int64, len(v))
	errs := make([]error, len(v))
	for idx, val := range v {
		r[idx], errs[idx] = strconv.ParseInt(val, 10, 64)
	}
	return r, errs
}
func StrToInt64(v string) int64 {
	if v, err := strconv.ParseInt(v, 10, 64); err == nil {
		return v
	}
	return 0
}
func StrToInt(v string) int {
	if v, err := strconv.Atoi(v); err == nil {
		return v
	}
	return 0
}
func StrToIntSlice(v []string) ([]int, []error) {
	r := make([]int, len(v))
	errs := make([]error, len(v))
	for idx, val := range v {
		r[idx], errs[idx] = strconv.Atoi(val)
	}
	return r, errs
}
func StrToFloat64Slice(v []string) ([]float64, []error) {
	r := make([]float64, len(v))
	errs := make([]error, len(v))
	for idx, val := range v {
		r[idx], errs[idx] = strconv.ParseFloat(val, 64)
	}
	return r, errs
}
func StrToFloat64SliceWithDefault(v []string, defaultVal float64) ([]float64, []error) {
	r := make([]float64, len(v))
	errs := make([]error, len(v))
	for idx, val := range v {
		if val == "" {
			r[idx] = defaultVal
			continue
		}
		r[idx], errs[idx] = strconv.ParseFloat(val, 64)
	}
	return r, errs
}
func StrToFloat64SliceWithDefaultZero(v []string) ([]float64, []error) {
	r := make([]float64, len(v))
	errs := make([]error, len(v))
	for idx, val := range v {
		if val == "" {
			r[idx] = 0
			continue
		}
		r[idx], errs[idx] = strconv.ParseFloat(val, 64)
	}
	return r, errs
}
func Float64ToStrSlice(v []float64) []string {
	r := make([]string, len(v))
	for idx, val := range v {
		r[idx] = strconv.FormatFloat(val, 'f', -1, 64)
	}
	return r
}
func Float64ToStrSliceWithDefault(v []float64, defaultVal float64) []string {
	r := make([]string, len(v))
	for idx, val := range v {
		if val == 0 {
			r[idx] = strconv.FormatFloat(defaultVal, 'f', -1, 64)
			continue
		}
		r[idx] = strconv.FormatFloat(val, 'f', -1, 64)
	}
	return r
}
func Float32ToStrSlice(v []float32) []string {
	r := make([]string, len(v))
	for idx, val := range v {
		r[idx] = strconv.FormatFloat(float64(val), 'f', -1, 32)
	}
	return r
}
func SplitStrFilterEmpty(v string, sep string) []string {
	r := strings.Split(v, sep)
	v1 := make([]string, 0)
	for idx, val := range r {
		if val != "" {
			v1 = append(v1, r[idx])
		}
	}
	return v1
}
func GetStructTargetFieldValue(data interface{}, targetFields []string) (fieldValues map[string]interface{}, successCount int) {
	fieldValues = make(map[string]interface{})
	if data == nil {
		return fieldValues, 0
	}
	var t reflect.Type
	var v reflect.Value
	dataType := reflect.ValueOf(data)
	if dataType.Kind() == reflect.Ptr {
		t = reflect.TypeOf(data).Elem()  // 获取指针类型的元素类型
		v = reflect.ValueOf(data).Elem() // 获取指针指向的值
	} else {
		t = reflect.TypeOf(data)
		v = reflect.ValueOf(data)
	}
	if dataType.Kind() != reflect.Struct {
		return fieldValues, 0
	}
	successCount = 0
	for _, fieldName := range targetFields {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			jsonTag := field.Tag.Get("json")
			if jsonTag == fieldName {
				fieldValue := v.Field(i)
				fieldValues[fieldName] = fieldValue.Interface()
				successCount++

			}
		}
	}

	return fieldValues, successCount
}
