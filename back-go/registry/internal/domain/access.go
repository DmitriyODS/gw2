package domain

/* Доступ к реестру.

   Реестр принадлежит ЧЕЛОВЕКУ (устройство заметок и диска), а коллеги и
   компании получают его шарингом. Уровней три, и они вложены друг в друга:

     view  — смотреть, выгружать в xlsx, печатать и искать по QR;
     edit  — плюс вести записи (создавать, править, удалять);
     admin — плюс менять структуру самого реестра.

   Владелец сильнее любого уровня и один может удалить реестр и раздать доступ.
   Эффективный уровень — ЛУЧШИЙ из личной шары и шар компаний, где человек
   состоит: если реестр раздан компании на просмотр, а лично мне на правку, я
   правлю. */

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

// Scope — область списка реестров (вкладки раздела).
const (
	ScopeAll     = "all"     // всё, к чему есть доступ
	ScopeMine    = "mine"    // созданные мной
	ScopeShared  = "shared"  // расшаренные мне лично
	ScopeCompany = "company" // расшаренные компаниям, где я состою
)

// NormalizeScope — область списка; незнакомая означает «всё».
func NormalizeScope(scope string) string {
	switch scope {
	case ScopeMine, ScopeShared, ScopeCompany:
		return scope
	default:
		return ScopeAll
	}
}
