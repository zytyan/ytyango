package mathparser

import "testing"

func benchmarkEvaluate(b *testing.B, expr string) {
	if _, err := Evaluate(expr); err != nil {
		b.Fatalf("warmup failed: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Evaluate(expr); err != nil {
			b.Fatalf("eval failed: %v", err)
		}
	}
}

func BenchmarkFastCheck(b *testing.B) {
	expr := "12+3*45-6/7+8.9"
	if !FastCheck(expr) {
		b.Fatal("unexpected false in warmup")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !FastCheck(expr) {
			b.Fatal("unexpected false result")
		}
	}
}

func BenchmarkEvaluateSimple(b *testing.B) {
	benchmarkEvaluate(b, "1+2*3-4/5+6")
}

func BenchmarkEvaluatePowFloat(b *testing.B) {
	benchmarkEvaluate(b, "1.02 ** 8.5 / pi + 3.5")
}

func BenchmarkEvaluateFactorial(b *testing.B) {
	benchmarkEvaluate(b, "20! / (10! * 2)")
}

func BenchmarkEvaluatePermutation(b *testing.B) {
	benchmarkEvaluate(b, "30P5")
}

func BenchmarkEvaluateParallelSimple(b *testing.B) {
	expr := "1+2*3-4/5+6"
	if _, err := Evaluate(expr); err != nil {
		b.Fatalf("warmup failed: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := Evaluate(expr); err != nil {
				b.Fatalf("eval failed: %v", err)
			}
		}
	})
}
