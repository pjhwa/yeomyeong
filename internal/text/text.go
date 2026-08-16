// Package text is the M1 system-message catalog (D-029).
// Locale ko is required. Locale en falls back to ko when the English
// string is missing. Default locale is ko.
package text

import "fmt"

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
)

// Sys codes for inventory (additive; WIRE-PROTOCOL M1 codes unchanged).
const (
	CodeNoExit      = "no_exit"
	CodeNotFound    = "not_found"
	CodeTooHeavy    = "too_heavy"
	CodeNotWearable = "not_wearable"
	CodeEmptySlot   = "empty_slot"
)

type entry struct {
	ko, en string
}

var catalog = map[string]entry{
	CmdUnknown:      {ko: "모르는 말입니다. say / quit"},
	MoveNoExit:      {ko: "그쪽으로는 갈 수 없습니다."},
	SysSeated:       {ko: "%s 님이 자리에 앉았습니다."},
	SysLeave:        {ko: "%s 님이 자리를 떴습니다."},
	SysRateLimit:    {ko: "rate_limited"},
	RoomExits:       {ko: "출구: %s"},
	RoomHere:        {ko: "여기: %s"},
	DirNorth:        {ko: "북쪽"},
	DirSouth:        {ko: "남쪽"},
	DirEast:         {ko: "동쪽"},
	DirWest:         {ko: "서쪽"},
	DirUp:           {ko: "위"},
	DirDown:         {ko: "아래"},
	RoomGround:      {ko: "바닥: %s"},
	GetMissing:      {ko: "여기에는 그것이 없습니다."},
	GetHeavy:        {ko: "너무 무거워 집을 수 없습니다."},
	GetOK:           {ko: "%s을(를) 집었습니다."},
	DropMissing:     {ko: "그런 물건이 없습니다."},
	DropOK:          {ko: "%s을(를) 내려놓았습니다."},
	EquipMissing:    {ko: "그런 물건이 없습니다."},
	EquipNotWear:    {ko: "그것은 착용할 수 없습니다."},
	EquipOK:         {ko: "%s을(를) 갖췄습니다."},
	UnequipEmpty:    {ko: "그 자리에는 아무것도 없습니다."},
	UnequipOK:       {ko: "%s을(를) 내려놓았습니다."},
	SheetInv:        {ko: "소지: %s"},
	SheetEquip:      {ko: "장비: %s"},
	SheetSkills:     {ko: "숙련: %s"},
	SheetStats:      {ko: "능력: %s"},
	SheetNone:       {ko: "없음"},
	SheetEquipSlots: {ko: "주손 %s, 몸 %s"},
	SheetTitle:      {ko: "호칭: %s"},
	PracticeGain:    {ko: "%s 숙련이 늘었습니다. (%d)"},
	PracticeMiss:    {ko: "아무것도 자리 잡지 않는다."},
	PracticeUnknown: {ko: "그런 숙련은 없습니다."},
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

// DirLabel is the localized compass word for a canonical dir, or dir itself.
func DirLabel(locale, dir string) string {
	if key, ok := dirKey[dir]; ok {
		return T(locale, key)
	}
	return dir
}
