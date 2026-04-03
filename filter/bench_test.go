package filter

import (
	"fmt"
	"testing"
)

// ---------------------------------------------------------------------------
// TextFilter.Matches benchmarks (hot path in column filtering)
// ---------------------------------------------------------------------------

func BenchmarkTextFilter_Matches(b *testing.B) {
	f := NewTextFilter()
	f.SetText("engineering")

	values := make([]any, 10_000)
	for i := range values {
		if i%10 == 0 {
			values[i] = "Engineering"
		} else {
			values[i] = fmt.Sprintf("Department_%d", i)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		for _, v := range values {
			f.Matches(v)
		}
	}
}

func BenchmarkTextFilter_Matches_Regex(b *testing.B) {
	f := NewTextFilter()
	f.SetRegex(true)
	f.SetText("^Eng")

	values := make([]any, 10_000)
	for i := range values {
		if i%10 == 0 {
			values[i] = "Engineering"
		} else {
			values[i] = fmt.Sprintf("Department_%d", i)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		for _, v := range values {
			f.Matches(v)
		}
	}
}

func BenchmarkTextFilter_Active(b *testing.B) {
	active := NewTextFilter()
	active.SetText("hello")
	inactive := NewTextFilter()

	b.Run("active", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			active.Active()
		}
	})
	b.Run("inactive", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			inactive.Active()
		}
	})
}

// ---------------------------------------------------------------------------
// NumberFilter.Matches benchmarks
// ---------------------------------------------------------------------------

func BenchmarkNumberFilter_Matches(b *testing.B) {
	f := NewNumberFilter()
	f.SetText(">50000")

	values := make([]any, 10_000)
	for i := range values {
		values[i] = float64(i) * 10
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		for _, v := range values {
			f.Matches(v)
		}
	}
}

func BenchmarkNumberFilter_Matches_Range(b *testing.B) {
	f := NewNumberFilter()
	f.SetText("10000..50000")

	values := make([]any, 10_000)
	for i := range values {
		values[i] = float64(i) * 10
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		for _, v := range values {
			f.Matches(v)
		}
	}
}

// ---------------------------------------------------------------------------
// SetFilter benchmarks (Issue #10 mentions SetFilter.Active() O(n) iteration)
// ---------------------------------------------------------------------------

func BenchmarkSetFilter_Active(b *testing.B) {
	for _, n := range []int{10, 100, 1_000} {
		b.Run(fmt.Sprintf("values=%d/none_excluded", n), func(b *testing.B) {
			vals := make([]string, n)
			for i := range vals {
				vals[i] = fmt.Sprintf("val_%d", i)
			}
			f := NewSetFilter(vals...)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				f.Active()
			}
		})

		b.Run(fmt.Sprintf("values=%d/one_excluded", n), func(b *testing.B) {
			vals := make([]string, n)
			for i := range vals {
				vals[i] = fmt.Sprintf("val_%d", i)
			}
			f := NewSetFilter(vals...)
			f.Exclude("val_0")
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				f.Active()
			}
		})
	}
}

func BenchmarkSetFilter_Matches(b *testing.B) {
	vals := make([]string, 100)
	for i := range vals {
		vals[i] = fmt.Sprintf("val_%d", i)
	}
	f := NewSetFilter(vals...)
	f.Exclude("val_50")

	values := make([]any, 10_000)
	for i := range values {
		values[i] = fmt.Sprintf("val_%d", i%100)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		for _, v := range values {
			f.Matches(v)
		}
	}
}
