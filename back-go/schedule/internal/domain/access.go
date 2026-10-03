package domain

/* Доступ к расписанию.

   Расписание лежит в пространстве: личном (принадлежит человеку) либо
   команды (принадлежит команде, автор лишь завёл его). Уровни вложены:

     view — смотреть, выгружать;
     edit — плюс вести занятия, категории, поля и настройки.

   «Владелец» сильнее любого уровня: удаляет расписание, раздаёт и переносит
   его между пространствами. У личного это хозяин, у расписания команды —
   автор и администраторы команды; рядовые участники получают team_access
   (по умолчанию просмотр: расписание обычно ведёт один человек). Адресная
   шара даёт только просмотр. */

const (
	AccessNone  = ""
	AccessView  = "view"
	AccessEdit  = "edit"
	AccessOwner = "owner"
)

var accessRank = map[string]int{AccessNone: 0, AccessView: 1, AccessEdit: 2, AccessOwner: 3}

// AccessAtLeast — уровень have не ниже нужного want.
func AccessAtLeast(have, want string) bool { return accessRank[have] >= accessRank[want] }

// NormalizeTeamAccess — уровень участников команды; незнакомое — просмотр.
func NormalizeTeamAccess(access string) string {
	if access == AccessEdit {
		return access
	}
	return AccessView
}
