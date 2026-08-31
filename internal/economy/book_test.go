package economy

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegionalPricesDiffer(t *testing.T) {
	b := NewBook([]Market{
		{ID: "cheap", Name: "싼 장터", Goods: map[string]Good{
			"herb": {Base: 3, Stock: 18, Target: 16, Demand: 0.7},
		}},
		{ID: "dear", Name: "비싼 장터", Goods: map[string]Good{
			"herb": {Base: 3, Stock: 4, Target: 6, Demand: 1.6},
		}},
	})
	pCheap, ok1 := b.Quote("cheap", "herb")
	pDear, ok2 := b.Quote("dear", "herb")
	if !ok1 || !ok2 {
		t.Fatalf("quote ok %v %v", ok1, ok2)
	}
	if pCheap >= pDear {
		t.Fatalf("want regional spread, cheap=%d dear=%d", pCheap, pDear)
	}
	if pCheap < 1 || pDear < 1 {
		t.Fatalf("min 1: %d %d", pCheap, pDear)
	}
}

func TestSellDropsPriceBuyRaises(t *testing.T) {
	b := NewBook([]Market{
		{ID: "m", Name: "장터", Goods: map[string]Good{
			"herb": {Base: 10, Stock: 8, Target: 8, Demand: 1},
		}},
	})
	before, _ := b.Quote("m", "herb")
	paid, ok := b.Sell("m", "herb", 4)
	if !ok || paid != before*4 {
		t.Fatalf("sell paid=%d ok=%v before=%d", paid, ok, before)
	}
	afterSell, _ := b.Quote("m", "herb")
	if afterSell >= before {
		t.Fatalf("sell must drop price: %d → %d", before, afterSell)
	}
	charged, ok := b.Buy("m", "herb", 4)
	if !ok {
		t.Fatal("buy")
	}
	afterBuy, _ := b.Quote("m", "herb")
	if charged != afterSell*4 {
		t.Fatalf("buy charged=%d unit=%d", charged, afterSell)
	}
	if afterBuy <= afterSell {
		t.Fatalf("buy must raise price: %d → %d", afterSell, afterBuy)
	}
	if _, ok := b.Buy("m", "herb", 100); ok {
		t.Fatal("over-buy")
	}
	if _, ok := b.Sell("m", "ghost", 1); ok {
		t.Fatal("ghost sell")
	}
}

func TestTickMeanReverts(t *testing.T) {
	b := NewBook([]Market{
		{ID: "m", Name: "장터", Goods: map[string]Good{
			"herb": {Base: 5, Stock: 2, Target: 6, Demand: 1},
			"ore":  {Base: 5, Stock: 9, Target: 6, Demand: 1},
		}},
	})
	b.Tick()
	if b.Stock("m", "herb") != 3 {
		t.Fatalf("herb stock=%d", b.Stock("m", "herb"))
	}
	if b.Stock("m", "ore") != 8 {
		t.Fatalf("ore stock=%d", b.Stock("m", "ore"))
	}
	for i := 0; i < 20; i++ {
		b.Tick()
	}
	if b.Stock("m", "herb") != 6 || b.Stock("m", "ore") != 6 {
		t.Fatalf("reverted herb=%d ore=%d", b.Stock("m", "herb"), b.Stock("m", "ore"))
	}
}

func TestPriceFloor(t *testing.T) {
	if p := Price(1, 0.1, 100, 1); p < 1 {
		t.Fatalf("floor %d", p)
	}
	if p := Price(10, 1, 8, 8); p != 10 {
		t.Fatalf("at target %d", p)
	}
}

func TestTradesAndList(t *testing.T) {
	b := NewBook([]Market{
		{ID: "a", Name: "에이", Goods: map[string]Good{"herb": {Base: 2, Stock: 1, Target: 1, Demand: 1}}},
		{ID: "b", Name: "비", Goods: map[string]Good{"ore": {Base: 2, Stock: 1, Target: 1, Demand: 1}}},
	})
	if !b.Trades("herb") || !b.Trades("ore") || b.Trades("nail") {
		t.Fatal("trades")
	}
	if !b.HasMarket("a") || b.HasMarket("no") {
		t.Fatal("has")
	}
	if b.MarketName("a") != "에이" || b.MarketName("no") != "no" {
		t.Fatal("name")
	}
	rows := b.List("a")
	if len(rows) != 1 || rows[0].ID != "herb" || rows[0].Price < 1 {
		t.Fatalf("list %+v", rows)
	}
	if NewBook(nil).List("a") != nil {
		t.Fatal("empty")
	}
}

func TestNoMutex(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(b, []byte("sync.Mutex")) || bytes.Contains(b, []byte("sync.RWMutex")) {
			t.Errorf("%s: economy book is loop-owned; no mutex", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
