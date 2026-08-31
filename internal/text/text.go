// Package text is the M1 system-message catalog (D-029).
// Locale ko is required. Locale en falls back to ko when the English
// string is missing. Default locale is ko.
package text

import (
	"fmt"
	"unicode/utf8"
)

const (
	LocaleKO = "ko"
	LocaleEN = "en"
	Default  = LocaleKO
)

// Keys for M0 system lines (wording unchanged) and M1 room chrome.
const (
	CmdUnknown      = "cmd.unknown"
	MoveNoExit      = "move.no_exit"
	SysSeated       = "sys.seated"
	SysLeave        = "sys.leave"
	SysRateLimit    = "sys.rate_limited"
	RoomExits       = "room.exits"
	RoomHere        = "room.here"
	DirNorth        = "dir.north"
	DirSouth        = "dir.south"
	DirEast         = "dir.east"
	DirWest         = "dir.west"
	DirUp           = "dir.up"
	DirDown         = "dir.down"
	RoomGround      = "room.ground"
	GetMissing      = "get.missing"
	GetHeavy        = "get.heavy"
	GetOK           = "get.ok"
	DropMissing     = "drop.missing"
	DropOK          = "drop.ok"
	EquipMissing    = "equip.missing"
	EquipNotWear    = "equip.not_wearable"
	EquipOK         = "equip.ok"
	UnequipEmpty    = "unequip.empty"
	UnequipOK       = "unequip.ok"
	SheetInv        = "sheet.inv"
	SheetEquip      = "sheet.equip"
	SheetSkills     = "sheet.skills"
	SheetStats      = "sheet.stats"
	SheetNone       = "sheet.none"
	SheetEquipSlots = "sheet.equip_slots"
	SheetTitle      = "sheet.title"
	PracticeGain    = "practice.gain"
	PracticeMiss    = "practice.miss"
	PracticeUnknown = "practice.unknown"
	SheetWallet     = "sheet.wallet"
	GatherNone      = "gather.none"
	GatherEmpty     = "gather.empty"
	GatherOK        = "gather.ok"
	GatherHeavy     = "gather.heavy"
	CraftWhat       = "craft.what"
	CraftNone       = "craft.none"
	CraftNeedRoom   = "craft.need_room"
	CraftNeedMat    = "craft.need_mat"
	CraftNeedTool   = "craft.need_tool"
	CraftOK         = "craft.ok"
	CraftHeavy      = "craft.heavy"
	CraftUnknown    = "craft.unknown"
	MarketNone      = "market.none"
	QuoteHeader     = "quote.header"
	QuoteLine       = "quote.line"
	QuoteHigh       = "quote.high"
	QuoteLow        = "quote.low"
	QuoteMid        = "quote.mid"
	SellNone        = "sell.none"
	SellMissing     = "sell.missing"
	SellShort       = "sell.short"
	SellOK          = "sell.ok"
	BuyNone         = "buy.none"
	BuyEmpty        = "buy.empty"
	BuyPoor         = "buy.poor"
	BuyOK           = "buy.ok"
	BuyHeavy        = "buy.heavy"
	TollPay         = "toll.pay"
	TollTake        = "toll.take"
)

// Sys codes for inventory (additive; WIRE-PROTOCOL M1 codes unchanged).
const (
	CodeNoExit      = "no_exit"
	CodeNotFound    = "not_found"
	CodeTooHeavy    = "too_heavy"
	CodeNotWearable = "not_wearable"
	CodeEmptySlot   = "empty_slot"
	CodeNoMarket    = "no_market"
	CodeNoStock     = "no_stock"
	CodeTooPoor     = "too_poor"
	CodeNoNode      = "no_node"
	CodeNoRecipe    = "no_recipe"
	CodeNeedMat     = "need_mat"
)

type entry struct {
	ko, en string
}

var catalog = map[string]entry{
	CmdUnknown:      {ko: "무슨 말인지 모르겠습니다. 보다, 종료"},
	MoveNoExit:      {ko: "그쪽으로는 갈 수 없습니다."},
	SysSeated:       {ko: "%s 님이 들어왔습니다."},
	SysLeave:        {ko: "%s 님이 나갔습니다."},
	SysRateLimit:    {ko: "너무 빨리 입력했습니다. 잠깐만 기다리세요."},
	RoomExits:       {ko: "출구: %s"},
	RoomHere:        {ko: "여기: %s"},
	DirNorth:        {ko: "북쪽"},
	DirSouth:        {ko: "남쪽"},
	DirEast:         {ko: "동쪽"},
	DirWest:         {ko: "서쪽"},
	DirUp:           {ko: "위"},
	DirDown:         {ko: "아래"},
	RoomGround:      {ko: "바닥: %s"},
	GetMissing:      {ko: "여기엔 그런 게 없습니다."},
	GetHeavy:        {ko: "너무 무거워서 집을 수 없습니다."},
	GetOK:           {ko: "%s%s 집었습니다."},
	DropMissing:     {ko: "그런 물건이 없습니다."},
	DropOK:          {ko: "%s%s 내려놓았습니다."},
	EquipMissing:    {ko: "그런 물건이 없습니다."},
	EquipNotWear:    {ko: "그건 들 수 없습니다."},
	EquipOK:         {ko: "%s%s 들었습니다."},
	UnequipEmpty:    {ko: "거기엔 아무것도 없습니다."},
	UnequipOK:       {ko: "%s%s 내려놓았습니다."},
	SheetInv:        {ko: "가진 것: %s"},
	SheetEquip:      {ko: "들고 있는 것: %s"},
	SheetSkills:     {ko: "기술: %s"},
	SheetStats:      {ko: "몸: %s"},
	SheetNone:       {ko: "없음"},
	SheetEquipSlots: {ko: "손 %s, 몸 %s"},
	SheetTitle:      {ko: "불리는 이름: %s"},
	PracticeGain:    {ko: "조금 익숙해진 것 같다."},
	PracticeMiss:    {ko: "아직 손에 안 익는다."},
	PracticeUnknown: {ko: "숙련할 기술 중에 그런 기술은 없습니다."},
	SheetWallet:     {ko: "주머니: %d냥"},
	GatherNone:      {ko: "여기서는 집을 게 없다."},
	GatherEmpty:     {ko: "지금은 더 없다. 조금 기다려야 한다."},
	GatherOK:        {ko: "%s%s 손에 넣었다."},
	GatherHeavy:     {ko: "너무 무거워서 더 못 넣는다."},
	CraftWhat:       {ko: "무엇을 만들까? %s"},
	CraftNone:       {ko: "여기서는 만들 게 없다."},
	CraftNeedRoom:   {ko: "%s에서 해야 한다."},
	CraftNeedMat:    {ko: "%s%s 더 구해야 한다."},
	CraftNeedTool:   {ko: "%s%s 들어야 한다."},
	CraftOK:         {ko: "%s%s 만들었다."},
	CraftHeavy:      {ko: "너무 무거워서 만들 수 없다."},
	CraftUnknown:    {ko: "그런 만드는 법은 모른다."},
	MarketNone:      {ko: "여기는 장터가 아니다."},
	QuoteHeader:     {ko: "%s 시세"},
	QuoteLine:       {ko: "%s %d냥 — %s"},
	QuoteHigh:       {ko: "오늘은 흔하다"},
	QuoteLow:        {ko: "품귀가 돌고 있다"},
	QuoteMid:        {ko: "값이 잠잠하다"},
	SellNone:        {ko: "여기는 그런 걸 받지 않는다."},
	SellMissing:     {ko: "그런 물건이 없다."},
	SellShort:       {ko: "그만큼은 없다."},
	SellOK:          {ko: "%s%s %d냥 받고 넘겼다."},
	BuyNone:         {ko: "그런 물건은 안 판다."},
	BuyEmpty:        {ko: "지금은 바닥이다."},
	BuyPoor:         {ko: "주머니가 모자란다."},
	BuyOK:           {ko: "%s%s %d냥 주고 샀다."},
	BuyHeavy:        {ko: "너무 무거워서 살 수 없다."},
	TollPay:         {ko: "검문에서 짐을 훑더니 통행세 %d냥을 걷어 갔다."},
	TollTake:        {ko: "검문에서 짐을 훑더니 %s%s 두고 가라고 했다."},
}

var dirKey = map[string]string{
	"north": DirNorth,
	"south": DirSouth,
	"east":  DirEast,
	"west":  DirWest,
	"up":    DirUp,
	"down":  DirDown,
}

// T looks up key for locale. Missing en uses ko. Missing key returns key.
func T(locale, key string, args ...any) string {
	e, ok := catalog[key]
	s := key
	if ok {
		s = e.ko
		if locale == LocaleEN && e.en != "" {
			s = e.en
		}
	}
	if len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, args...)
}

// EulReul is 을 or 를 after a Korean word (everyday object particle).
func EulReul(word string) string {
	if hangulBatchim(word) {
		return "을"
	}
	return "를"
}

func hangulBatchim(word string) bool {
	r, _ := utf8.DecodeLastRuneInString(word)
	return r >= '가' && r <= '힣' && (r-'가')%28 != 0
}

// DirLabel is the localized compass word for a canonical dir, or dir itself.
func DirLabel(locale, dir string) string {
	if key, ok := dirKey[dir]; ok {
		return T(locale, key)
	}
	return dir
}
