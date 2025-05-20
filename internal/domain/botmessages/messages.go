package botmessages

const (
	MsgUnknownCommand  = "Неизвестная команда. Попробуйте ввести заново!"
	MsgNoSavedPages    = "Нет отслеживаемых ссылок"
	MsgGotNoLink       = "Не передано ни одной ссылки"
	MsgWrongFormatLink = `Вы передали ссылку неверного формата. Ссылки, которые я могу сохранить, имеют вид:
https://stackoverflow.com/questions/id_of_question/answers
https://stackoverflow.com/questions/id_of_question/comments
https://github.com/author/repository/pulls
https://github.com/author/repository/issues
`
	MsgNoTags             = "Не передано ни одного тега"
	MsgNoTag              = "Не передан тег"
	MsgTooManyTags        = "Я могу удалять только один тег за раз. Введите команду заново."
	MsgNoSavedPagesByTag  = "Нет ссылок с таким тегом"
	MsgNoSavedPagesByTags = "Нет ссылок с такими тегами"
	MsgTagDeleteFailed    = "Не удалось удалить тег"
	MsgLinkNotFound       = "В сохранённых нет такой ссылки"
	MsgErrAddLink         = "Произошла ошибка при сохранении ссылки"
	MsgErrDeleteLink      = "Произошла ошибка при удалении ссылки"
	MsgSaved              = "Сохранил!"
	MsgDeleted            = "Удалил!"
	MsgLinkAlreadyExists  = "В списке отслеживаемых уже есть эта ссылка "
	MsgAddTags            = "Введите теги через пробел"
	MsgAddFilters         = "Введите фильтры через пробел"
)

const MsgHelp = `Я могу сохранять твои ссылки для отслеживания. 
Если хочешь начать отслеживать изменения по ссылке, отправь мне её в формате /track ссылка.
Чтобы прекратить отслеживание ссылки, отправь /untrack ссылка. 
Чтобы просмотреть все отслеживаемые ссылки, отправь /list,
а если хочешь просмотреть ссылки  только с определёнными тегами - отправь /listbytags список тегов через пробел.
Чтобы удалить тег, воспользуйся командой /deletetag тег.
`

const MsgHello = "Добро пожаловать! 👾\n\n" + MsgHelp
