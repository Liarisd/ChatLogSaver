----- ENGLISH LANG -----

# Chat Log Saver
**Chat Log Saver** is a lightweight Windows utility designed to automatically preserve client logs from **Neverwinter Nights: Enhanced Edition (NWN:EE)**.
NWN:EE can overwrite its client log files when the game is restarted. Chat Log Saver helps preserve previous sessions by creating timestamped copies of the original log files.
The application runs quietly in the Windows system tray and requires no interaction during gameplay.

## Features
- **Automatic log monitoring** - detects when the NWN:EE client is running and checks log files approximately once per second.
- **Automatic folder detection** - locates the NWN:EE log directory in the user's Windows Documents folder.
- **Manual folder selection** - allows the user to select the logs directory if the default location is unavailable.
- **Timestamped copies** - saves original TXT files with the system date and time in their filenames.
- **Unmodified content** - preserves the original log contents without conversion, filtering, or formatting.
- **Background operation** - runs from the Windows system tray.
- **Windows startup** - can be enabled or disabled directly from the tray menu.
- **Automatic language selection** - Russian interface when the Windows UI language is Russian; English for all other languages.
- **Built-in CLS icon** - no external icon file is required.
- **Standalone executable** - no Python, PowerShell scripts, or additional runtime installation required.

## Installation
1. Download the Windows ZIP archive from the repository's [Releases](https://github.com/Liarisd/ChatLogSaver/releases) page.
2. Extract `ChatLogSaver.exe` to a folder of your choice.
3. Run `ChatLogSaver.exe`.
4. Find the CLS icon in the Windows system tray.
5. Right-click the icon to open the application menu.

You can enable automatic startup with Windows from the tray menu.

## How It Works
By default, the application looks for log files in:
`Documents\Neverwinter Nights\logs\`
Saved copies are stored in:
`Documents\Neverwinter Nights\logs\saved\`
Example filenames:
```text
nwclientLog1_2026-10-24_19-30-00.txt
nwclientLog2_2026-10-24_19-30-00.txt
```
The timestamp represents the start of the detected game session.
While NWN:EE is running, Chat Log Saver checks the source files approximately once per second and updates their saved copies when changes are detected.
If the logs folder cannot be found, the **Select logs folder...** option becomes available in the tray menu. The selected path is remembered for future launches.

## System Requirements
- Windows 10 or Windows 11 (64-bit)
- Neverwinter Nights: Enhanced Edition
- No additional runtime dependencies

## Building from Source
The application is written in **Go 1.22+**, using native Windows APIs.
Build with:

```bash
GOOS=windows GOARCH=amd64 go build -ldflags="-H=windowsgui -s -w" -o ChatLogSaver.exe ./src
```

The CLS tray icon is embedded into the executable.

## Limitations
- Windows version only.
- The application must be running to monitor game sessions.
- It can preserve only data that NWN:EE has already written to disk.
- Log preservation during crashes or very rapid restarts is best-effort and cannot be guaranteed.
- The executable is not digitally signed, so Windows SmartScreen may display a warning.

## About
Developed for **Tauredor | Edge of the Woods**, a persistent world for Neverwinter Nights: Enhanced Edition.

Website: [tauredor.ru](https://tauredor.ru)
Source code: [GitHub](https://github.com/Liarisd/ChatLogSaver)

----- RUSSIAN LANG -----

# Chat Log Saver
**Chat Log Saver** — небольшая фоновая утилита для автоматического сохранения журналов клиента **Neverwinter Nights: Enhanced Edition (NWN:EE)**.
NWN:EE может перезаписывать клиентские журналы при повторном запуске игры. Chat Log Saver позволяет сохранять историю предыдущих игровых сессий, создавая копии исходных файлов с датой и временем.
Программа работает в системном трее Windows и не требует участия игрока во время игры.

## Возможности
- **Автоматическое отслеживание журналов** - определяет запуск клиента NWN:EE и проверяет изменения файлов примерно раз в секунду.
- **Автоматический поиск папки** - находит каталоги журналов NWN:EE в пользовательской папке «Документы».
- **Ручной выбор папки** - позволяет указать расположение журналов, если стандартный каталог отсутствует.
- **Сохранение с датой и временем** - создаёт копии исходных TXT с временной отметкой в имени файла.
- **Без изменения содержимого** - не преобразует, не фильтрует и не форматирует журналы.
- **Фоновая работа** - размещается в системном трее Windows.
- **Автозапуск** - включается и отключается непосредственно через меню программы.
- **Русский и английский языки** - интерфейс автоматически выбирается по языку Windows. Для всех языков, кроме русского, используется английский.
- **Встроенная иконка CLS** - отдельный файл значка не требуется.
- **Самостоятельный EXE** - не требует Python, PowerShell-скриптов или дополнительных библиотек.

## Установка
1. Скачайте ZIP-архив для Windows из раздела [Releases](https://github.com/Liarisd/ChatLogSaver/releases).
2. Распакуйте `ChatLogSaver.exe` в удобную папку.
3. Запустите `ChatLogSaver.exe`.
4. Найдите значок CLS в системном трее Windows.
5. Нажмите на него правой кнопкой мыши для открытия меню.

При необходимости включите автоматический запуск программы вместе с Windows.

## Принцип работы
По умолчанию программа ищет журналы в каталоге:
`Документы\Neverwinter Nights\logs\`
Копии сохраняются в подпапке:
`Документы\Neverwinter Nights\logs\saved\`
Пример сохранённых файлов:
```text
nwclientLog1_2026-10-24_19-30-00.txt
nwclientLog2_2026-10-24_19-30-00.txt
```
Дата и время в названии соответствуют началу обнаруженной игровой сессии.
Во время работы NWN:EE утилита проверяет файлы примерно раз в секунду и обновляет сохранённые копии при обнаружении изменений.
Если папка журналов не найдена, в меню становится доступен пункт **«Выбрать папку логов...»**. Выбранный путь сохраняется и используется при последующих запусках программы.

## Системные требования
- Windows 10 или Windows 11 (64-bit)
- Neverwinter Nights: Enhanced Edition
- Дополнительное программное обеспечение не требуется

## Сборка из исходников
Программа написана на **Go 1.22+** с использованием штатного Windows API.
Команда сборки:
```bash
GOOS=windows GOARCH=amd64 go build -ldflags="-H=windowsgui -s -w" -o ChatLogSaver.exe ./src
```
Иконка CLS для системного трея встроена в исполняемый файл.

## Ограничения
- Текущая версия предназначена для Windows.
- Для отслеживания игровых сессий программа должна быть запущена.
- Сохраняются только данные, которые клиент NWN:EE уже записал на диск.
- При аварийном завершении и очень быстром перезапуске клиента сохранение всех последних записей не гарантируется.
- EXE не подписан цифровым сертификатом издателя, поэтому Windows SmartScreen может отображать предупреждение.

## О проекте
Программа разработана для постоянного мира **Tauredor | Край Лесов** на базе Neverwinter Nights: Enhanced Edition.
Сайт проекта: [tauredor.ru](https://tauredor.ru)
Исходный код: [GitHub](https://github.com/Liarisd/ChatLogSaver)
