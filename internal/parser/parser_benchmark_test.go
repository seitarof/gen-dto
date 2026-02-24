package parser

import "testing"

func BenchmarkParseRecursive_User(b *testing.B) {
	p := New()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		infos, err := p.ParseRecursive("github.com/seitarof/gen-dto/testdata/bidi/source", "User")
		if err != nil {
			b.Fatal(err)
		}
		if len(infos) == 0 {
			b.Fatal("empty parse result")
		}
	}
}
