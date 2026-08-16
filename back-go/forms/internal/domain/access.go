package domain

/* Доступ к форме.

   Форма принадлежит ЧЕЛОВЕКУ (устройство реестров, заметок и диска), а коллеги
   и компании получают её адресно. Уровней три, и они вложены друг в друга:

     respond — заполнить форму (это же и «назначение»: у адресата появляется
               обязанность ответить, а у автора — контроль исполнения);
     view    — плюс видеть ответы, сводку и выгрузку;
     edit    — плюс менять саму форму и раздавать доступ.

   Владелец сильнее любого уровня и один может форму удалить. Эффективный
   уровень — ЛУЧШИЙ из личной шары и шар компаний, где человек состоит. */

const (
	AccessNone    = ""
	AccessRespond = "respond"
	AccessView    = "view"
	AccessEdit    = "edit"
	AccessOwner   = "owner"
)

// accessRank — порядок уровней для сравнения «не ниже чем». Задан числом, а не
// сравнением строк: 'view' > 'edit' лексикографически, и наивный MAX(access)
// молча понижал бы права. Держать в паре с accessExpr репозитория.
var accessRank = map[string]int{
	AccessNone: 0, AccessRespond: 1, AccessView: 2, AccessEdit: 3, AccessOwner: 4,
}

// AccessAtLeast — уровень have не ниже нужного want.
func AccessAtLeast(have, want string) bool { return accessRank[have] >= accessRank[want] }

// BestAccess — сильнейший из уровней (доступ приходит несколькими путями сразу).
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
// значение трактуем как «заполнить»: ошибиться в сторону меньших прав
// безопаснее. Владельцем поделиться нельзя — это принадлежность, а не уровень.
func NormalizeShareAccess(access string) string {
	switch access {
	case AccessView, AccessEdit:
		return access
	default:
		return AccessRespond
	}
}

// Scope — область списка форм (вкладки раздела).
const (
	ScopeAll      = "all"      // всё, к чему есть доступ
	ScopeMine     = "mine"     // свои
	ScopeAssigned = "assigned" // мне назначили заполнить (лично или моей компании)
	ScopeShared   = "shared"   // совместные: мне открыли ответы или правку
)

// NormalizeScope — область списка; незнакомая означает «всё».
func NormalizeScope(scope string) string {
	switch scope {
	case ScopeMine, ScopeAssigned, ScopeShared:
		return scope
	default:
		return ScopeAll
	}
}

// NormalizeStatus — состояние приёма ответов; незнакомое означает черновик.
func NormalizeStatus(status string) string {
	switch status {
	case StatusOpen, StatusClosed:
		return status
	default:
		return StatusDraft
	}
}

// NormalizeQuizRelease — когда показывать оценку теста.
func NormalizeQuizRelease(v string) string {
	if v == QuizManual {
		return QuizManual
	}
	return QuizImmediately
}
