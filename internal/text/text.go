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
	CmdUnknown   = "cmd.unknown"
	MoveNoExit   = "move.no_exit"
	SysSeated    = "sys.seated"
	SysLeave     = "sys.leave"
	SysRateLimit = "sys.rate_limited"
	RoomExits    = "room.exits"
	RoomHere     = "room.here"
	DirNorth     = "dir.north"
	DirSouth     = "dir.south"
	DirEast      = "dir.east"
	DirWest      = "dir.west"
	DirUp        = "dir.up"
	DirDown      = "dir.down"
)

// CodeNoExit is the WIRE-PROTOCOL sys code for a blocked move.
const CodeNoExit = "no_exit"

type entry struct {
	ko, en string
}

var catalog = map[string]entry{
	CmdUnknown:   {ko: "모르는 말입니다. say / quit"},
	MoveNoExit:   {ko: "그쪽으로는 갈 수 없습니다."},
	SysSeated:    {ko: "%s 님이 자리에 앉았습니다."},
	SysLeave:     {ko: "%s 님이 자리를 떴습니다."},
	SysRateLimit: {ko: "rate_limited"},
	RoomExits:    {ko: "출구: %s"},
	RoomHere:     {ko: "여기: %s"},
	DirNorth:     {ko: "북쪽"},
	DirSouth:     {ko: "남쪽"},
	DirEast:      {ko: "동쪽"},
	DirWest:      {ko: "서쪽"},
	DirUp:        {ko: "위"},
	DirDown:      {ko: "아래"},
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
