# Chat Log Saver (Windows)

Небольшая фоновая утилита для сохранения неизменённых файлов `nwclientLog*.txt` клиента Neverwinter Nights: Enhanced Edition.

## Запуск

1. Запустите `ChatLogSaver.exe`. Дополнительные программы, Python или VBS не требуются.
2. Правой кнопкой мыши на значке CLS в трее открывается меню.
3. Выберите «Включить/отключить автозапуск», если нужен запуск при входе в Windows.

Логи сохраняются в подпапку `saved` каталога `Neverwinter Nights\logs` в пользовательских «Документах». Программа проверяет клиент и файлы раз в секунду и хранит копии с системной датой и временем начала сеанса в названии. Содержимое файлов не преобразует.

## Сборка из исходников
Нужен Go 1.22 или новее:

```sh
GOOS=windows GOARCH=amd64 go build -ldflags="-H=windowsgui -s -w" -o ChatLogSaver.exe ./src
```
Иконка CLS встроена в приложение с помощью `go:embed`, поэтому для работы трея рядом с EXE не требуется файл ICO. В Проводнике значок EXE может быть стандартным: отдельный PE-ресурс значка не добавлен.
Программа собрана кросс-компиляцией; реальное поведение необходимо проверить на Windows. EXE не подписан цифровой подписью издателя.

## Interface language / Язык интерфейса
Chat Log Saver selects the interface language automatically when it starts:
- Russian Windows UI language (`GetUserDefaultUILanguage`, primary language ID `0x19`) → Russian.
- All other Windows UI languages → English.

No translation files or manual language settings are required. Restart the application after changing the Windows UI language.
Программа определяет язык интерфейса Windows при запуске: русский → русский, любой другой → английский. Для применения изменений языка Windows перезапустите программу.

### Автозапуск и выбор папки
- Меню автоматически показывает **«Включить автозапуск» / «Enable startup with Windows»**, если запись ChatLogSaver отсутствует в `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, либо **«Выключить автозапуск» / «Disable startup with Windows»**, если она существует.
- Программа ищет журналы в системной папке `Documents\Neverwinter Nights\logs` (включая перенаправленную папку «Документы»).
- Если папка не существует, становится доступен пункт **«Выбрать папку логов...» / «Select logs folder...»**. Выбирать нужно саму папку `logs`. После выбора путь сохраняется в пользовательской папке конфигурации Windows: `ChatLogSaver\logs-path.txt` и используется при следующих запусках.
- Если папка найдена, пункт выбора папки отключён.
- Содержимое логов не преобразуется.
