package domain

/* Доступ к реестру.

   Реестр лежит в пространстве: личном (принадлежит человеку) либо команды
   (принадлежит команде, автор лишь завёл его). Коллеги и команды получают
   доступ ещё и шарингом. Уровней три, и они вложены друг в друга:

     view  — смотреть, выгружать в xlsx, печатать и искать по QR;
     edit  — плюс вести записи (создавать, править, удалять);
     admin — плюс менять структуру самого реестра.

   «Владелец» сильнее любого уровня: удаляет реестр, переносит его между
   пространствами и раздаёт доступ. У личного реестра это его хозяин, у реестра
   команды — автор и администраторы команды; рядовые участники получают
   уровень team_access. Эффективный уровень — ЛУЧШИЙ из всех путей: если
   команда видит реестр на просмотр, а лично мне его дали на правку, я правлю. */

const (
	AccessNone  = ""
	AccessView  = "view"
	AccessEdit  = "edit"
	AccessAdmin = "admin"
	AccessOwner = "owner"
)

// accessRank — порядок уровней для сравнения «не ниже чем».
var accessRank = map[string]int{
	AccessNone: 0, AccessView: 1, AccessEdit: 2, AccessAdmin: 3, AccessOwner: 4,
}

// AccessAtLeast — уровень have не ниже нужного want.
func AccessAtLeast(have, want string) bool { return accessRank[have] >= accessRank[want] }

// BestAccess — сильнейший из уровней (человеку доступ может прийти сразу
// несколькими путями).
func BestAccess(levels ...string) string {
	best := AccessNone
	for _, l := range levels {
		if accessRank[l] > accessRank[best] {
			best = l
		}
	}
	return best
}

// NormalizeShareAccess — привести выданный уровень к допустимому. Незнакомое
// значение трактуем как просмотр: ошибиться в сторону меньших прав безопаснее.
// Владельцем поделиться нельзя — это не уровень доступа, а принадлежность.
func NormalizeShareAccess(access string) string {
	switch access {
	case AccessEdit, AccessAdmin:
		return access
	default:
		return AccessView
	}
}

// NormalizeTeamAccess — уровень участников команды. Владение раздать нельзя,
// незнакомое значение — правка (умолчание для вещи команды).
func NormalizeTeamAccess(access string) string {
	switch access {
	case AccessView, AccessAdmin:
		return access
	default:
		return AccessEdit
	}
}

// Scope — область списка реестров (вкладки раздела).
const (
	ScopeAll    = "all"    // всё, к чему есть доступ
	ScopeMine   = "mine"   // личное пространство
	ScopeTeam   = "team"   // пространства команд, где я состою
	ScopeShared = "shared" // расшаренные мне лично или моим командам
)

// NormalizeScope — область списка; незнакомая означает «всё». Прежнее имя
// «company» понимается как команды.
func NormalizeScope(scope string) string {
	switch scope {
	case ScopeMine, ScopeTeam, ScopeShared:
		return scope
	case "company":
		return ScopeTeam
	default:
		return ScopeAll
	}
}
