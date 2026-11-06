package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/soulteary/warden/internal/define"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkObservedSingleSource(b *testing.B) {
	for _, rows := range []int{1000, 10000} {
		b.Run(fmt.Sprint(rows), func(b *testing.B) {
			users := make([]define.AllowListUser, rows)
			for i := range users {
				users[i] = define.AllowListUser{Mail: fmt.Sprintf("member%d@example.com", i), UserID: fmt.Sprint(i), Status: "active"}
			}
			raw, err := json.Marshal(users)
			if err != nil {
				b.Fatal(err)
			}
			file := filepath.Join(b.TempDir(), "rules.json")
			if err = os.WriteFile(file, raw, 0o600); err != nil {
				b.Fatal(err)
			}
			loader, err := newTestRulesLoader(nil, ModeOnlyLocal)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.SetBytes(int64(len(raw)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result := loader.LoadWithResult(context.Background(), file, "", "", "")
				if result.Err != nil || len(result.Users) != rows {
					b.Fatalf("load failed: %v", result.Err)
				}
			}
		})
	}
}
