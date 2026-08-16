package skill

import (
	"math"
	"testing"
)

func always() float64 { return 0 }
func never() float64  { return 0.999999 }

func seq(vals ...float64) func() float64 {
	i := 0
	return func() float64 {
		if i >= len(vals) {
			return 1
		}
		v := vals[i]
		i++
		return v
	}
}

func TestChanceAndDifficulty(t *testing.T) {
	const rank, diff = 50, 50
	d := float64(rank-diff) / 18
	want := 0.55 * math.Exp(-0.5*d*d) / (1 + float64(rank)/12)
	if math.Abs(Chance(rank, diff)-want) > 1e-12 {
		t.Fatalf("formula %g want %g", Chance(rank, diff), want)
	}
	if Chance(RankCap, 0) != 0 || Chance(-3, 0) != Chance(0, 0) {
		t.Fatal("cap/clamp")
	}
	matched, easy := Chance(rank, rank), Chance(rank, rank-25)
	if matched <= easy || easy > 0.05 {
		t.Fatalf("matched=%g easy=%g", matched, easy)
	}
	cat := loadOK(t)
	s := cat.NewSheet().WithRank("test:swing", rank)
	if g, _ := s.TryGain("test:swing", rank, func() float64 { return matched - 1e-9 }); !g {
		t.Fatal("matched should gain")
	}
	if g, _ := s.TryGain("test:swing", rank-25, func() float64 { return matched - 1e-9 }); g {
		t.Fatal("easy should miss")
	}
	p := Chance(90, 0)
	if p > 1e-5 {
		t.Fatalf("falloff p=%g", p)
	}
	high := cat.NewSheet().WithRank("test:swing", 90)
	if g, out := high.TryGain("test:swing", 0, func() float64 { return p }); g || out.Rank("test:swing") != 90 {
		t.Fatalf("p≈0 gained=%v", g)
	}
}

func TestTryGainCaps(t *testing.T) {
	cat := loadOK(t)
	s := cat.NewSheet().WithRank("test:swing", 50).WithRank("pad", 650)
	gained, out := s.TryGain("test:swing", 50, always)
	if gained || out.Rank("test:swing") != 50 || out.RankSum() != RankSumCap {
		t.Fatalf("700 block %v %d %d", gained, out.Rank("test:swing"), out.RankSum())
	}
	under := cat.NewSheet().WithRank("test:swing", 50).WithRank("pad", 649)
	gained, out = under.TryGain("test:swing", 50, seq(0, 1))
	if !gained || out.Rank("test:swing") != 51 || out.RankSum() != RankSumCap {
		t.Fatalf("699 %v %d", gained, out.Rank("test:swing"))
	}
	full := cat.NewSheet().WithRank("test:swing", RankCap)
	if g, _ := full.TryGain("test:swing", RankCap, always); g {
		t.Fatal("rank 100")
	}

	empty := cat.NewSheet()
	gained, out = empty.TryGain("test:swing", 0, seq(0, 1))
	if !gained || out.Rank("test:swing") != 1 || empty.Rank("test:swing") != 0 {
		t.Fatalf("gain %v %d", gained, out.Rank("test:swing"))
	}
	if g, _ := empty.TryGain("test:swing", 0, never); g {
		t.Fatal("miss")
	}
	if g, _ := empty.TryGain("nope", 0, always); g {
		t.Fatal("unknown")
	}
	if g, _ := (Sheet{}).TryGain("test:swing", 0, always); g {
		t.Fatal("unbound")
	}
}

func TestStatSideEffect(t *testing.T) {
	cat := loadOK(t)
	s := cat.NewSheet()
	gained, out := s.TryGain("test:swing", 0, seq(0, 0))
	if !gained || out.Stat(statStr) != 1 {
		t.Fatalf("stat gain %v %d", gained, out.Stat(statStr))
	}
	gained, out = s.TryGain("test:swing", 0, seq(0, 1))
	if !gained || out.Stat(statStr) != 0 {
		t.Fatalf("stat miss %v %d", gained, out.Stat(statStr))
	}
	cap := cat.NewSheet().WithStat(statStr, StatCap)
	gained, out = cap.TryGain("test:swing", 0, always)
	if !gained || out.Stat(statStr) != StatCap {
		t.Fatal("stat 100")
	}
	summed := cat.NewSheet().
		WithStat(statStr, 50).WithStat(statDex, 50).WithStat(statVit, 50).
		WithStat(statWit, 50).WithStat(statSense, 50).WithStat(statFame, 50)
	gained, out = summed.TryGain("test:swing", 0, always)
	if !gained || out.Stat(statStr) != 50 || out.StatSum() != StatSumCap {
		t.Fatalf("stat sum 300 %d", out.StatSum())
	}
	edge := cat.NewSheet().WithStat(statStr, 50).WithStat(statDex, StatSumCap-51)
	gained, out = edge.TryGain("test:swing", 0, always)
	if !gained || out.Stat(statStr) != 51 || out.StatSum() != StatSumCap {
		t.Fatalf("stat 299 %d", out.Stat(statStr))
	}
	z := cat.NewSheet().WithRank("test:swing", 4).WithStat(statStr, 3)
	z = z.WithRank("test:swing", 0).WithStat(statStr, 0)
	if z.RankSum() != 0 || z.StatSum() != 0 {
		t.Fatal("zero delete")
	}
}

func TestDifficultyAndBind(t *testing.T) {
	if Difficulty(50, true) != 50 || Difficulty(50, false) != 25 || Difficulty(10, false) != 0 || Difficulty(-3, true) != 0 {
		t.Fatalf("diff %d %d %d %d", Difficulty(50, true), Difficulty(50, false), Difficulty(10, false), Difficulty(-3, true))
	}
	cat := loadOK(t)
	s := cat.Bind(map[string]int{"test:swing": 4, "gone": 0}, map[string]int{statStr: 3})
	if s.Rank("test:swing") != 4 || s.Stat(statStr) != 3 || s.Rank("gone") != 0 {
		t.Fatalf("bind %+v", s.Ranks())
	}
	ranks, stats := s.Ranks(), s.Stats()
	ranks["hacked"] = 9
	stats[statStr] = 99
	if s.Rank("hacked") != 0 || s.Stat(statStr) != 3 {
		t.Fatal("mutated")
	}
	if Always() != 0 || Never() < 1 {
		t.Fatal("hooks")
	}
}

func TestDefaultRandAndText(t *testing.T) {
	for i := 0; i < 8; i++ {
		if v := DefaultRand(); v < 0 || v >= 1 {
			t.Fatalf("rng %g", v)
		}
	}
	if _, out := loadOK(t).NewSheet().TryGain("test:swing", 0, nil); out.cat == nil {
		t.Fatal("nil rng")
	}
	l := Localized{KO: "호", EN: "Title"}
	if l.Text("en") != "Title" || l.Text("ko") != "호" || (Localized{KO: "호"}).Text("en") != "호" {
		t.Fatalf("%+v", l)
	}
}
