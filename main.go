package main

import (
	_ "embed"
	"io"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/atotto/clipboard"
	"github.com/getlantern/systray"
	"golang.org/x/sys/windows"
)

var (
	currentIP string
	interval  = 10 * time.Minute
	services  = []string{
		"https://checkip.amazonaws.com",
		"https://icanhazip.com",
		"https://ipinfo.io/ip",
		"https://ifconfig.me/ip",
	}
)

//go:embed icon.ico
var iconData []byte

func alreadyRunning() bool {
	// создаём глобальный mutex
	name := "Global\\IPCheckerUniqueName"
	h, err := windows.CreateMutex(nil, false, windows.StringToUTF16Ptr(name))
	if err != nil {
		return true
	}

	// проверяем существует ли уже mutex
	lastErr := syscall.GetLastError()
	if lastErr == syscall.ERROR_ALREADY_EXISTS {
		return true
	}

	// сохраняем дескриптор чтобы не закрыть сразу
	_ = h
	return false
}

func showMessage(title, message string) {
	user32 := windows.NewLazySystemDLL("user32.dll")
	messageBox := user32.NewProc("MessageBoxW")

	titlePtr, _ := windows.UTF16PtrFromString(title)
	messagePtr, _ := windows.UTF16PtrFromString(message)

	messageBox.Call(0, uintptr(unsafe.Pointer(messagePtr)), uintptr(unsafe.Pointer(titlePtr)), 0)
}

func main() {
	if alreadyRunning() {
		showMessage("Error", "IP Checker has already been launched!")
		os.Exit(0)
	}
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(iconData)
	systray.SetTitle("IP Checker")
	systray.SetTooltip("Getting an IP address...")

	mCopy := systray.AddMenuItem("Copy IP", "Copy the current IP")
	mUpdate := systray.AddMenuItem("Refresh", "Refresh IP")
	mInterval := systray.AddMenuItem("Interval", "IP check interval")
	m1 := mInterval.AddSubMenuItem("1 minute", "")
	m5 := mInterval.AddSubMenuItem("5 minutes", "")
	m10 := mInterval.AddSubMenuItem("10 minutes", "")
	m30 := mInterval.AddSubMenuItem("30 minutes", "")
	mQuit := systray.AddMenuItem("Exit", "Close program")

	updateIP()
	go func() {
		for {
			time.Sleep(interval)
			updateIP()
		}
	}()

	go func() {
		for {
			select {
			case <-mUpdate.ClickedCh:
				updateIP()
			case <-mCopy.ClickedCh:
				if currentIP != "" {
					clipboard.WriteAll(currentIP)
				}
			case <-m1.ClickedCh:
				interval = 1 * time.Minute
			case <-m5.ClickedCh:
				interval = 5 * time.Minute
			case <-m10.ClickedCh:
				interval = 10 * time.Minute
			case <-m30.ClickedCh:
				interval = 30 * time.Minute
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {}

func updateIP() {
	for _, url := range services {
		resp, err := http.Get(url)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}
		ip := strings.TrimSpace(string(body))
		if ip != "" {
			currentIP = ip
			// Tooltip = полный IP при наведении
			systray.SetTooltip(currentIP)
			systray.SetTitle("IP Checker")
			return
		}
	}
	systray.SetTitle("No access")
	systray.SetTooltip("There is no access to the services")
}
