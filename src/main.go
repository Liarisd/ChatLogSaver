// Chat Log Saver for Windows. Standard library only; no PowerShell or VBS.
package main

import (
	_ "embed"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	wmDestroy           = 0x0002
	wmClose             = 0x0010
	wmCommand           = 0x0111
	wmTimer             = 0x0113
	wmRButtonUp         = 0x0205
	wmContextMenu       = 0x007b
	wmNull              = 0x0000
	wmLButtonDblClk     = 0x0203
	wmApp               = 0x8000
	wmTray              = wmApp + 1
	wmUpdateStatus      = wmApp + 2
	nidAdd              = 0
	nidDelete           = 2
	nifMessage          = 1
	nifIcon             = 2
	nifTip              = 4
	wsOverlapped        = 0
	mfString            = 0
	mfSeparator         = 0x800
	mfGray              = 1
	tpmRightButton      = 2
	tpmReturnCmd        = 0x0100
	wsPopup             = 0x80000000
	wsExToolWindow      = 0x00000080
	swShowNoActivate    = 4
	tpmBottomAlign      = 0x20
	imageIcon           = 1
	lrLoadFromFile      = 0x0010
	lrDefaultSize       = 0x0040
	idOpen              = 101
	idSource            = 102
	idAutostart         = 103
	idQuit              = 104
	idStatus            = 105
	idChooseFolder      = 106
	bifReturnOnlyFSDirs = 0x0001
	bifNewDialogStyle   = 0x0040
)

type point struct{ X, Y int32 }
type browseInfo struct {
	Owner       uintptr
	Root        uintptr
	DisplayName uintptr
	Title       uintptr
	Flags       uint32
	Callback    uintptr
	Param       uintptr
	Image       int32
}
type msg struct {
	HWND           uintptr
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             point
	Private        uint32
}
type wndClass struct {
	Size                                                               uint32
	Style                                                              uint32
	WndProc                                                            uintptr
	ClsExtra, WndExtra                                                 int32
	Instance, Icon, Cursor, Background, MenuName, ClassName, SmallIcon uintptr
}
type notifyIcon struct {
	Size             uint32
	HWND             uintptr
	ID               uint32
	Flags            uint32
	CallbackMessage  uint32
	Icon             uintptr
	Tip              [128]uint16
	State, StateMask uint32
	Info             [256]uint16
	Timeout          uint32
	InfoTitle        [64]uint16
	InfoFlags        uint32
	Item             uintptr
	Guid             [16]byte
	BalloonIcon      uintptr
}

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	shell32             = syscall.NewLazyDLL("shell32.dll")
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	pRegister           = user32.NewProc("RegisterClassExW")
	pCreate             = user32.NewProc("CreateWindowExW")
	pDef                = user32.NewProc("DefWindowProcW")
	pGetMessage         = user32.NewProc("GetMessageW")
	pTranslate          = user32.NewProc("TranslateMessage")
	pDispatch           = user32.NewProc("DispatchMessageW")
	pLoadIcon           = user32.NewProc("LoadIconW")
	pLoadImage          = user32.NewProc("LoadImageW")
	pCreateIconResource = user32.NewProc("CreateIconFromResourceEx")
	pShowWindow         = user32.NewProc("ShowWindow")
	pPostMessage        = user32.NewProc("PostMessageW")
	pCreateMenu         = user32.NewProc("CreatePopupMenu")
	pAppendMenu         = user32.NewProc("AppendMenuW")
	pTrackMenu          = user32.NewProc("TrackPopupMenu")
	pDestroyMenu        = user32.NewProc("DestroyMenu")
	pSetForeground      = user32.NewProc("SetForegroundWindow")
	pGetCursor          = user32.NewProc("GetCursorPos")
	pPostQuit           = user32.NewProc("PostQuitMessage")
	pDestroyWnd         = user32.NewProc("DestroyWindow")
	pSetTimer           = user32.NewProc("SetTimer")
	pShellNotify        = shell32.NewProc("Shell_NotifyIconW")
	pShellExecute       = shell32.NewProc("ShellExecuteW")
	pGetModule          = kernel32.NewProc("GetModuleHandleW")
	pGetUILanguage      = kernel32.NewProc("GetUserDefaultUILanguage")
	pBrowseFolder       = shell32.NewProc("SHBrowseForFolderW")
	pGetPathFromIDList  = shell32.NewProc("SHGetPathFromIDListW")
	pCoTaskMemFree      = syscall.NewLazyDLL("ole32.dll").NewProc("CoTaskMemFree")
	russianUI           bool
	hwnd                uintptr
	ni                  notifyIcon
	sourceDir, savedDir string
	sessions            = map[string]*session{}
	lastErr             string
	running             = false
	lastStatus          = ""
	stateMu             sync.RWMutex
	stopWorker          = make(chan struct{})
	startTime           = time.Now()
	menuShowing         bool
	lastMenuTime        time.Time
)

// Windows language: Russian primary language ID is 0x19; all others use English.
func detectRussianUI() bool {
	lang, _, _ := pGetUILanguage.Call()
	return lang&0x03ff == 0x19
}
func localized(ru, en string) string {
	if russianUI {
		return ru
	}
	return en
}

type session struct {
	start        time.Time
	targets      map[string]string
	fingerprints map[string]string
}

func u(s string) *uint16   { return syscall.StringToUTF16Ptr(s) }
func ptr(s string) uintptr { return uintptr(unsafe.Pointer(u(s))) }
func showFolder(path string) {
	os.MkdirAll(path, 0755)
	pShellExecute.Call(0, ptr("open"), ptr("explorer.exe"), ptr("\""+path+"\""), 0, 1)
}

func folderExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
func customPathFile() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "ChatLogSaver", "logs-path.txt")
}
func loadCustomPath() string {
	config := customPathFile()
	if config == "" {
		return ""
	}
	raw, err := os.ReadFile(config)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}
func saveCustomPath(path string) error {
	config := customPathFile()
	if config == "" {
		return fmt.Errorf("config location unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(config), 0700); err != nil {
		return err
	}
	return os.WriteFile(config, []byte(path), 0600)
}
func chooseLogFolder() {
	if folderExists(sourceDir) {
		return
	}
	var display [260]uint16
	bi := browseInfo{Owner: hwnd, DisplayName: uintptr(unsafe.Pointer(&display[0])), Title: ptr(localized("Выберите папку logs Neverwinter Nights", "Select the Neverwinter Nights logs folder")), Flags: bifReturnOnlyFSDirs | bifNewDialogStyle}
	pidl, _, _ := pBrowseFolder.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return
	}
	defer pCoTaskMemFree.Call(pidl)
	var path [260]uint16
	ok, _, _ := pGetPathFromIDList.Call(pidl, uintptr(unsafe.Pointer(&path[0])))
	if ok == 0 {
		return
	}
	chosen := syscall.UTF16ToString(path[:])
	if !folderExists(chosen) {
		return
	}
	if err := saveCustomPath(chosen); err != nil {
		stateMu.Lock()
		lastErr = err.Error()
		stateMu.Unlock()
		return
	}
	stateMu.Lock()
	sourceDir = chosen
	savedDir = filepath.Join(chosen, "saved")
	// Keep the existing session tied to its initial archive filenames only if it is active;
	// initialize a new session to avoid carrying source-file fingerprints from an old location.
	if running {
		sessions["current"] = &session{start: time.Now(), targets: map[string]string{}, fingerprints: map[string]string{}}
	}
	lastErr = ""
	stateMu.Unlock()
}

//go:embed cls.ico
var embeddedICO []byte

// The icon is held in the executable image. No adjacent .ico file is needed.
func loadAppIcon() uintptr {
	if len(embeddedICO) < 22 || binary.LittleEndian.Uint16(embeddedICO[2:4]) != 1 {
		return 0
	}
	count := int(binary.LittleEndian.Uint16(embeddedICO[4:6]))
	chosen := -1
	for i := 0; i < count; i++ {
		pos := 6 + i*16
		if pos+16 > len(embeddedICO) {
			break
		}
		// Windows tray: prefer 32x32 to retain clear detail on HiDPI monitors.
		w := int(embeddedICO[pos])
		if w == 0 {
			w = 256
		}
		if chosen == -1 || w == 32 {
			chosen = pos
		}
		if w == 32 {
			break
		}
	}
	if chosen < 0 {
		return 0
	}
	size := int(binary.LittleEndian.Uint32(embeddedICO[chosen+8 : chosen+12]))
	offset := int(binary.LittleEndian.Uint32(embeddedICO[chosen+12 : chosen+16]))
	if size == 0 || offset < 0 || offset+size > len(embeddedICO) {
		return 0
	}
	data := embeddedICO[offset : offset+size]
	h, _, _ := pCreateIconResource.Call(uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), 1, 0x00030000, 0, 0, 0)
	runtime.KeepAlive(data)
	return h
}

func createTarget(name string, start time.Time) string {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	stamp := start.Format("2006-01-02_15-04-05")
	for i := 0; i < 10000; i++ {
		suffix := ""
		if i > 0 {
			suffix = fmt.Sprintf("_%d", i)
		}
		path := filepath.Join(savedDir, base+"_"+stamp+suffix+filepath.Ext(name))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path
		}
	}
	return filepath.Join(savedDir, base+"_"+stamp+"_extra"+filepath.Ext(name))
}
func fileCopy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + fmt.Sprintf(".%d.tmp", time.Now().UnixNano())
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if err == nil {
		err = out.Sync()
	}
	closeErr := out.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	// Windows MoveFileExW with REPLACE_EXISTING: atomic replacement on the same volume.
	oldName := u(tmp)
	newName := u(dst)
	proc := kernel32.NewProc("MoveFileExW")
	r, _, e := proc.Call(uintptr(unsafe.Pointer(oldName)), uintptr(unsafe.Pointer(newName)), 1|8)
	if r == 0 {
		os.Remove(tmp)
		return fmt.Errorf("замена копии: %v", e)
	}
	return nil
}
func saveSession(s *session, force bool) {
	files, err := filepath.Glob(filepath.Join(sourceDir, "nwclientLog*.txt"))
	if err != nil {
		lastErr = err.Error()
		return
	}
	if len(files) == 0 {
		return
	}
	if err = os.MkdirAll(savedDir, 0755); err != nil {
		lastErr = err.Error()
		return
	}
	for _, f := range files {
		fi, err := os.Stat(f)
		if err != nil || fi.IsDir() || fi.ModTime().Before(s.start.Add(-3*time.Second)) {
			continue
		}
		name := filepath.Base(f)
		key := strings.ToLower(name)
		fingerprint := fmt.Sprintf("%d/%d", fi.Size(), fi.ModTime().UnixNano())
		dst, ok := s.targets[key]
		if !ok {
			dst = createTarget(name, s.start)
			s.targets[key] = dst
		}
		if force || !ok || s.fingerprints[key] != fingerprint {
			if err := fileCopy(f, dst); err != nil {
				lastErr = err.Error()
			} else {
				s.fingerprints[key] = fingerprint
			}
		}
	}
}

// Get-Process replacement via Toolhelp snapshot: use only documented Windows APIs.
type processEntry struct {
	Size, Cnt, PID                  uint32
	DefaultHeap                     uintptr
	ModuleID, CntThreads, ParentPID int32
	PriorityBase                    int32
	Flags                           uint32
	Exe                             [260]uint16
}

func nwnRunning() bool {
	snapProc := kernel32.NewProc("CreateToolhelp32Snapshot")
	first := kernel32.NewProc("Process32FirstW")
	next := kernel32.NewProc("Process32NextW")
	closeHandle := kernel32.NewProc("CloseHandle")
	h, _, _ := snapProc.Call(2, 0)
	if h == ^uintptr(0) {
		return false
	}
	defer closeHandle.Call(h)
	var e processEntry
	e.Size = uint32(unsafe.Sizeof(e))
	r, _, _ := first.Call(h, uintptr(unsafe.Pointer(&e)))
	for r != 0 {
		if strings.EqualFold(syscall.UTF16ToString(e.Exe[:]), "nwmain.exe") {
			return true
		}
		r, _, _ = next.Call(h, uintptr(unsafe.Pointer(&e)))
	}
	return false
}

// tick runs on a background goroutine, never from the Win32 message handler.
// Disk flushes and NWN file reads must not block tray clicks.
func tick() {
	active := nwnRunning()
	stateMu.Lock()
	if active && !running {
		sessions["current"] = &session{start: time.Now(), targets: map[string]string{}, fingerprints: map[string]string{}}
	}
	if s := sessions["current"]; s != nil {
		saveSession(s, !active)
		if !active {
			delete(sessions, "current")
		}
	}
	running = active
	status := localized("Ожидание NWN", "Waiting for NWN")
	if active {
		status = localized("NWN запущен", "NWN running")
	}
	if lastErr != "" {
		status = localized("Ошибка записи", "Write error")
	}
	changed := status != lastStatus
	lastStatus = status
	stateMu.Unlock()
	if changed && hwnd != 0 {
		pPostMessage.Call(hwnd, wmUpdateStatus, 0, 0)
	}
}
func logWorker() {
	tick()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			tick()
		case <-stopWorker:
			return
		}
	}
}

func setTooltip(t string) {
	for i := range ni.Tip {
		ni.Tip[i] = 0
	}
	copy(ni.Tip[:], syscall.StringToUTF16(t))
	ni.Flags = nifMessage | nifIcon | nifTip
	pShellNotify.Call(1, uintptr(unsafe.Pointer(&ni)))
}
func addMenu(m uintptr, flags, id uintptr, label string) { pAppendMenu.Call(m, flags, id, ptr(label)) }
func doCommand(id uintptr) {
	switch id {
	case idOpen:
		stateMu.RLock()
		path := savedDir
		exists := folderExists(sourceDir)
		stateMu.RUnlock()
		if exists {
			showFolder(path)
		}
	case idSource:
		stateMu.RLock()
		path := sourceDir
		exists := folderExists(sourceDir)
		stateMu.RUnlock()
		if exists {
			showFolder(path)
		}
	case idChooseFolder:
		chooseLogFolder()
	case idAutostart:
		startupToggle()
	case idQuit:
		pDestroyWnd.Call(hwnd)
	}
}
func trayMenu() {
	if menuShowing || time.Since(lastMenuTime) < 250*time.Millisecond {
		return
	}
	menuShowing = true
	defer func() { menuShowing = false; lastMenuTime = time.Now() }()
	m, _, _ := pCreateMenu.Call()
	if m == 0 {
		return
	}
	defer pDestroyMenu.Call(m)
	stateMu.RLock()
	status := localized("Ожидание NWN", "Waiting for NWN")
	if running {
		status = localized("NWN запущен", "NWN running")
	}
	if lastErr != "" {
		status = localized("Ошибка: ", "Error: ") + lastErr
		if len([]rune(status)) > 70 {
			status = string([]rune(status)[:70])
		}
	}
	sourceMissing := !folderExists(sourceDir)
	stateMu.RUnlock()
	if sourceMissing {
		status = localized("Папка логов не найдена", "Logs folder not found")
	}
	addMenu(m, mfString|mfGray, idStatus, status)
	addMenu(m, mfSeparator, 0, "")
	if sourceMissing {
		addMenu(m, mfString|mfGray, idOpen, localized("Открыть сохранённые логи", "Open saved logs"))
		addMenu(m, mfString|mfGray, idSource, localized("Открыть исходные логи", "Open original logs"))
		addMenu(m, mfString, idChooseFolder, localized("Выбрать папку логов...", "Select logs folder..."))
	} else {
		addMenu(m, mfString, idOpen, localized("Открыть сохранённые логи", "Open saved logs"))
		addMenu(m, mfString, idSource, localized("Открыть исходные логи", "Open original logs"))
		addMenu(m, mfString|mfGray, idChooseFolder, localized("Выбрать папку логов...", "Select logs folder..."))
	}
	if startupEnabled() {
		addMenu(m, mfString, idAutostart, localized("Выключить автозапуск", "Disable startup with Windows"))
	} else {
		addMenu(m, mfString, idAutostart, localized("Включить автозапуск", "Enable startup with Windows"))
	}
	addMenu(m, mfSeparator, 0, "")
	addMenu(m, mfString, idQuit, localized("Выход", "Exit"))
	var p point
	pGetCursor.Call(uintptr(unsafe.Pointer(&p)))
	// Unlike a permanently hidden overlapped window, a tiny visible toolwindow can
	// receive foreground activation. This prevents intermittent menu failures.
	pShowWindow.Call(hwnd, swShowNoActivate)
	pSetForeground.Call(hwnd)
	chosen, _, _ := pTrackMenu.Call(m, tpmRightButton|tpmBottomAlign|tpmReturnCmd, uintptr(p.X), uintptr(p.Y), 0, hwnd, 0)
	// Microsoft-recommended WM_NULL after TrackPopupMenu for notification icons.
	pPostMessage.Call(hwnd, wmNull, 0, 0)
	pShowWindow.Call(hwnd, 0)
	if chosen != 0 {
		doCommand(chosen)
	}
}
func startupEnabled() bool {
	key, err := syscall.UTF16PtrFromString(`Software\Microsoft\Windows\CurrentVersion\Run`)
	if err != nil {
		return false
	}
	var h syscall.Handle
	err = syscall.RegOpenKeyEx(syscall.HKEY_CURRENT_USER, key, 0, syscall.KEY_QUERY_VALUE, &h)
	if err != nil {
		return false
	}
	defer syscall.RegCloseKey(h)
	name := u("ChatLogSaver")
	var typ, n uint32
	return syscall.RegQueryValueEx(h, name, nil, &typ, nil, &n) == nil
}
func startupToggle() {
	// HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run (no admin required)
	key, err := syscall.UTF16PtrFromString(`Software\Microsoft\Windows\CurrentVersion\Run`)
	if err != nil {
		return
	}
	var h syscall.Handle
	err = syscall.RegOpenKeyEx(syscall.HKEY_CURRENT_USER, key, 0, syscall.KEY_QUERY_VALUE|syscall.KEY_SET_VALUE, &h)
	if err != nil {
		lastErr = err.Error()
		return
	}
	defer syscall.RegCloseKey(h)
	name := u("ChatLogSaver")
	var typ, n uint32
	err = syscall.RegQueryValueEx(h, name, nil, &typ, nil, &n)
	if err == nil {
		proc := syscall.NewLazyDLL("advapi32.dll").NewProc("RegDeleteValueW")
		r, _, _ := proc.Call(uintptr(h), uintptr(unsafe.Pointer(name)))
		if r != 0 {
			lastErr = fmt.Sprintf("RegDeleteValue: %d", r)
		}
	} else {
		exe, e := os.Executable()
		if e != nil {
			lastErr = e.Error()
			return
		}
		v := syscall.StringToUTF16("\"" + exe + "\"")
		bytes := unsafe.Slice((*byte)(unsafe.Pointer(&v[0])), len(v)*2)
		proc := syscall.NewLazyDLL("advapi32.dll").NewProc("RegSetValueExW")
		r, _, _ := proc.Call(uintptr(h), uintptr(unsafe.Pointer(name)), 0, 1, uintptr(unsafe.Pointer(&bytes[0])), uintptr(len(bytes)))
		if r != 0 {
			lastErr = fmt.Sprintf("RegSetValueEx: %d", r)
		}
	}
}
func windowProc(w uintptr, message uint32, wp, lp uintptr) uintptr {
	switch message {
	case wmTray:
		if lp == wmRButtonUp || lp == wmContextMenu {
			trayMenu()
		} else if lp == wmLButtonDblClk {
			showFolder(savedDir)
		}
		return 0
	case wmUpdateStatus:
		stateMu.RLock()
		status := lastStatus
		stateMu.RUnlock()
		setTooltip("Chat Log Saver - " + status)
		return 0
	case wmCommand:
		doCommand(wp & 0xffff)
		return 0
	case wmClose:
		pDestroyWnd.Call(hwnd)
		return 0
	case wmDestroy:
		close(stopWorker)
		pShellNotify.Call(nidDelete, uintptr(unsafe.Pointer(&ni)))
		pPostQuit.Call(0)
		return 0
	}
	r, _, _ := pDef.Call(w, uintptr(message), wp, lp)
	return r
}
func main() {
	runtime.LockOSThread() // Window, message loop and popup menu must use one OS thread.
	russianUI = detectRussianUI()
	defer runtime.UnlockOSThread()
	docs, err := os.UserHomeDir()
	if err != nil {
		return
	}
	sourceDir = filepath.Join(docs, "Documents", "Neverwinter Nights", "logs")
	// Documents may be redirected (OneDrive, custom paths). Query Windows known folder via SHGetFolderPathW CSIDL_PERSONAL.
	var path [260]uint16
	getFolder := shell32.NewProc("SHGetFolderPathW")
	r, _, _ := getFolder.Call(0, 5, 0, 0, uintptr(unsafe.Pointer(&path[0])))
	if r == 0 && path[0] != 0 {
		sourceDir = filepath.Join(syscall.UTF16ToString(path[:]), "Neverwinter Nights", "logs")
	}
	if custom := loadCustomPath(); folderExists(custom) {
		sourceDir = custom
	}
	savedDir = filepath.Join(sourceDir, "saved")
	inst, _, _ := pGetModule.Call(0)
	className := u("ChatLogSaverHiddenWindow")
	ico := loadAppIcon()
	if ico == 0 {
		ico, _, _ = pLoadIcon.Call(0, 32512)
	}
	cb := syscall.NewCallback(windowProc)
	wc := wndClass{Size: uint32(unsafe.Sizeof(wndClass{})), WndProc: cb, Instance: inst, ClassName: uintptr(unsafe.Pointer(className)), Icon: ico, SmallIcon: ico}
	ret, _, _ := pRegister.Call(uintptr(unsafe.Pointer(&wc)))
	if ret == 0 {
		return
	}
	hwnd, _, _ = pCreate.Call(wsExToolWindow, uintptr(unsafe.Pointer(className)), ptr("Chat Log Saver"), wsPopup, ^uintptr(0)-31999, ^uintptr(0)-31999, 1, 1, 0, 0, inst, 0)
	if hwnd == 0 {
		return
	}
	ni = notifyIcon{Size: uint32(unsafe.Sizeof(notifyIcon{})), HWND: hwnd, ID: 1, Flags: nifMessage | nifIcon | nifTip, CallbackMessage: wmTray, Icon: ico}
	copy(ni.Tip[:], syscall.StringToUTF16("Chat Log Saver"))
	pShellNotify.Call(nidAdd, uintptr(unsafe.Pointer(&ni)))
	go logWorker()
	var m msg
	for {
		r, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslate.Call(uintptr(unsafe.Pointer(&m)))
		pDispatch.Call(uintptr(unsafe.Pointer(&m)))
	}
}
