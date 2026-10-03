package domain

/* Доступ к ежедневнику.

   Ежедневник лежит в пространстве: личном (принадлежит человеку) либо
   команды (принадлежит команде, автор лишь завёл его). Уровни вложены:

     view  — смотреть записи и выгружать;
     check — плюс отмечать записи выполненными (сценарий «руководитель
             раздаёт дела, сотрудник закрывает»);
     edit  — плюс вести записи: создавать, править, переносить, удалять.

   «Владелец» сильнее любого уровня: переименовывает и удаляет ежедневник,
   раздаёт доступ и переносит его между пространствами. У личного это его
   хозяин, у ежедневника команды — автор и администраторы команды; рядовые
   участники получают team_access. Адресная шара даёт view или check. */

const (
	AccessNone  = ""
	AccessView  = "view"
	AccessCheck = "check"
	AccessEdit  = "edit"
	AccessOwner = "owner"
)

var accessRank = map[string]int{
	AccessNone: 0, AccessView: 1, AccessCheck: 2, AccessEdit: 3, AccessOwner: 4,
}

// AccessAtLeast — уровень have не ниже нужного want.
func AccessAtLeast(have, want string) bool { return accessRank[have] >= accessRank[want] }

// NormalizeTeamAccess — уровень участников команды; незнакомое значение —
// правка (умолчание для вещи команды).
func NormalizeTeamAccess(access string) string {
	switch access {
	case AccessView, AccessCheck:
		return access
	default:
		return AccessEdit
	}
}

// Виды ежедневника.
const (
	KindRegular = "regular"
	// KindMyDay — скрытый личный ежедневник экрана «Сегодня»: один на
	// человека, в разделе «Ежедневники» не показывается, записи в нём бывают
	// без даты («Потом»).
	KindMyDay = "my_day"
)
