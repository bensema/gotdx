package gotdx

import (
	"testing"

	"github.com/bensema/gotdx/proto"
)

// TestSortMACTickChartDaysAsc 验证多日分时序列按交易日升序重排：服务端倒序
// （新到旧）会被纠正为旧到新，已升序的保持不变，日期为空的条目统一排到末尾。
func TestSortMACTickChartDaysAsc(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"服务端倒序重排为升序", []string{"2026-11-10", "2026-11-09", "2026-11-06"},
			[]string{"2026-11-06", "2026-11-09", "2026-11-10"}},
		{"已是升序保持不变", []string{"2026-11-06", "2026-11-09"},
			[]string{"2026-11-06", "2026-11-09"}},
		{"空日期排到末尾", []string{"", "2026-11-10", "2026-11-09"},
			[]string{"2026-11-09", "2026-11-10", ""}},
		{"跨年跨月按字典序即时间序", []string{"2026-01-05", "2025-12-31", "2026-02-02"},
			[]string{"2025-12-31", "2026-01-05", "2026-02-02"}},
		{"单元素", []string{"2026-11-10"}, []string{"2026-11-10"}},
		{"空切片", nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			days := make([]proto.MACTickChartDay, len(tc.in))
			for i, d := range tc.in {
				// PreClose 复用为「原始序号」标记，便于在稳定排序用例中断言顺序。
				days[i] = proto.MACTickChartDay{Date: d, PreClose: float64(i)}
			}
			sortMACTickChartDaysAsc(days)
			if len(days) != len(tc.want) {
				t.Fatalf("len = %d, want %d", len(days), len(tc.want))
			}
			for i, w := range tc.want {
				if days[i].Date != w {
					t.Fatalf("idx %d date = %q, want %q（实际全序列 %v）",
						i, days[i].Date, w, datesOf(days))
				}
			}
		})
	}
}

// TestSortMACTickChartDaysAscStable 验证相同日期保持服务端原有相对顺序，
// 空日期排在所有正常日期之后。
func TestSortMACTickChartDaysAscStable(t *testing.T) {
	days := []proto.MACTickChartDay{
		{Date: "2026-11-10", PreClose: 1},
		{Date: "2026-11-09", PreClose: 2},
		{Date: "2026-11-10", PreClose: 3},
		{Date: "", PreClose: 4},
		{Date: "2026-11-09", PreClose: 5},
	}
	sortMACTickChartDaysAsc(days)
	want := []float64{2, 5, 1, 3, 4}
	for i, w := range want {
		if days[i].PreClose != w {
			t.Fatalf("idx %d = %v, want %v（实际全序列 %v）", i, days[i].PreClose, w, datesOf(days))
		}
	}
}

// datesOf 提取序列中的日期文本，仅用于测试失败时的可读输出。
func datesOf(days []proto.MACTickChartDay) []string {
	out := make([]string, len(days))
	for i, d := range days {
		out[i] = d.Date
	}
	return out
}
