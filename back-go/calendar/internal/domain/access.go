package domain

/* Доступ к календарю.

   Календарь лежит в пространстве: личном (принадлежит человеку) либо команды
   (принадлежит команде, автор лишь завёл его). Уровней три, и они вложены:

     view  — смотреть события, выгружать;
     edit  — плюс вести записи;
     admin — плюс менять структуру: название, поля, внешние ссылки.

   «Владелец» сильнее любого уровня: удаляет календарь и переносит его между
   пространствами. У личного календаря это его хозяин, у календаря команды —
   автор и администраторы команды; рядовые участники получают team_access. */

const (
	AccessNone  = ""
	AccessView  = "view"
	AccessEdit  = "edit"
	AccessAdmin = "admin"
	AccessOwner = "owner"
)

var accessRank = map[string]int{
	AccessNone: 0, AccessView: 1, AccessEdit: 2, AccessAdmin: 3, AccessOwner: 4,
}

// AccessAtLeast — уровень have не ниже нужного want.
func AccessAtLeast(have, want string) bool { return accessRank[have] >= accessRank[want] }

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
