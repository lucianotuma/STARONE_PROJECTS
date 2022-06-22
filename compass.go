package main

//TODO TRANSFORMAR O Ctrl+C em exit

import (
	"encoding/csv"
	"fmt"
	"image/png"
	"log"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/kbinani/screenshot"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	user32_Must             = syscall.MustLoadDLL("user32.dll")
	kernel32                = syscall.MustLoadDLL("kernel32.dll")
	procGetAsyncKeyState    = user32.NewProc("GetAsyncKeyState")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow") //GetForegroundWindow
	procGetWindowTextW      = user32.NewProc("GetWindowTextW")      //GetWindowTextW
	getLastInputInfo        = user32_Must.MustFindProc("GetLastInputInfo")
	getTickCount            = kernel32.MustFindProc("GetTickCount")
	lastInputInfo           struct {
		cbSize uint32
		dwTime uint32
	}
	tmpTitle string
	//get windows login user name
	userName = os.Getenv("USERNAME")
	//get ip address

	//get mac address

)

const (
	keyloggerSleepTime      time.Duration = 150 * time.Millisecond
	idleTimeSleepTime       time.Duration = 200 * time.Millisecond
	idleTimeRegisterBuffer  time.Duration = 20 * time.Second
	takeScreenshotSleepTime time.Duration = 1 * time.Minute
	windowLoggerSleepTime   time.Duration = 100 * time.Millisecond
	ipAddress               string        = "127.0.0.1"
	macAddress              string        = "00:00:00:00:00:00"
)

func realizaRegistro(registros [][]string, arquivoCsv string) {
	f, err := os.OpenFile(arquivoCsv, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalln("failed to open file", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	for _, record := range registros {
		if err := w.Write(record); err != nil {
			log.Fatalln("error writing record to file", err)
		}
	}
}

func getForegroundWindow() (hwnd syscall.Handle, err error) {
	r0, _, e1 := syscall.Syscall(procGetForegroundWindow.Addr(), 0, 0, 0, 0)
	if e1 != 0 {
		err = error(e1)
		return
	}
	hwnd = syscall.Handle(r0)
	return
}

func idleTime() {
	var a time.Duration = 0
	defer realizaRegistro([][]string{{macAddress, ipAddress, userName, fmt.Sprint(a), time.Now().Format("01.02.2006-15.04.05")}}, "idleTime.csv")
	for {
		time.Sleep(idleTimeSleepTime)
		lastInputInfo.cbSize = uint32(unsafe.Sizeof(lastInputInfo))
		currentTickCount, _, _ := getTickCount.Call()
		r1, _, err := getLastInputInfo.Call(uintptr(unsafe.Pointer(&lastInputInfo)))
		if r1 == 0 {
			panic("error getting last input info: " + err.Error())
		}
		b := time.Duration((uint32(currentTickCount) - lastInputInfo.dwTime)) * time.Millisecond
		if a > b && a > idleTimeRegisterBuffer {
			fmt.Println("IdleTime: ", a)
			go realizaRegistro([][]string{{macAddress, ipAddress, userName, fmt.Sprint(a), time.Now().Format("01.02.2006-15.04.05")}}, "idleTime.csv")
			a = 0
		} else {
			a = b
		}
	}
}

func getWindowText(hwnd syscall.Handle, str *uint16, maxCount int32) (len int32, err error) {
	r0, _, e1 := syscall.Syscall(procGetWindowTextW.Addr(), 3, uintptr(hwnd), uintptr(unsafe.Pointer(str)), uintptr(maxCount))
	len = int32(r0)
	if len == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func takeScreenshot() {
	for {
		time.Sleep(takeScreenshotSleepTime)
		n := screenshot.NumActiveDisplays()
		for i := 0; i < n; i++ {
			bounds := screenshot.GetDisplayBounds(i)
			img, err := screenshot.CaptureRect(bounds)
			if err != nil {
				panic(err)
			}
			currentTime := time.Now()
			fileName := fmt.Sprintf("%s_%d_%dx%d_%s.png", userName, i, bounds.Dx(), bounds.Dy(), currentTime.Format("01.02.2006-15.04.05"))
			file, _ := os.Create(fileName)
			png.Encode(file, img)
			fmt.Println("Screenshot taked:", fileName)
			file.Close()
		}
	}
}

func windowLogger() {
	timeVariable := time.Now()
	logRegistroDuracaoTela := make([][]string, 0)
	defer realizaRegistro(logRegistroDuracaoTela, "duracaoTela.csv")
	for {
		g, _ := getForegroundWindow()
		b := make([]uint16, 200)
		_, err := getWindowText(g, &b[0], int32(len(b)))
		if err != nil {
		}
		if syscall.UTF16ToString(b) != "" {
			if tmpTitle != syscall.UTF16ToString(b) {
				logRegistroDuracaoTela = append(logRegistroDuracaoTela, []string{macAddress, ipAddress, userName, tmpTitle, fmt.Sprint(time.Since(timeVariable)), time.Now().Format("01.02.2006-15.04.05")})
				timeVariable = time.Now()
				tmpTitle = syscall.UTF16ToString(b)
				fmt.Println("Criado log de mudança de tela:", logRegistroDuracaoTela)
				if len(logRegistroDuracaoTela) > 10 {
					go realizaRegistro(logRegistroDuracaoTela, "duracaoTela.csv")
					logRegistroDuracaoTela = nil
				}
			}
			time.Sleep(windowLoggerSleepTime)
		}
	}
}

func keyLogger() {
	i := 0
	defer realizaRegistro([][]string{{macAddress, ipAddress, userName, fmt.Sprint(i), time.Now().Format("01.02.2006-15.04.05")}}, "teclasApertadas.csv")
	for {
		time.Sleep(keyloggerSleepTime)
		for KEY := 0; KEY <= 256; KEY++ {
			Val, _, _ := procGetAsyncKeyState.Call(uintptr(KEY))
			if Val > 1 && KEY > 20 && KEY < 127 {
				i += 1
				fmt.Println("Contador de Teclas:", i)
				if i > 200 {
					go realizaRegistro([][]string{{macAddress, ipAddress, userName, fmt.Sprint(i), time.Now().Format("01.02.2006-15.04.05")}}, "teclasApertadas.csv")
					fmt.Println("Registrado teclas em csv e apagado contador:", i)
					i = 0
				}
			}
		}
	}
}

func main() {
	fmt.Println("Starting KeyLogger!")

	go takeScreenshot()
	go idleTime()
	go keyLogger()
	go windowLogger()
	fmt.Println("Logging")
	fmt.Println("Configuration:")
	fmt.Println("Configuration:")
	fmt.Println("keyloggerSleepTime:", keyloggerSleepTime)
	fmt.Println("idleTimeSleepTime:", idleTimeSleepTime)
	fmt.Println("TakeScreenshotSleepTime:", takeScreenshotSleepTime)
	fmt.Println("idleTimeRegisterBuffer:", idleTimeRegisterBuffer)
	fmt.Println("windowLoggerSleepTime:", windowLoggerSleepTime)
	for {
		time.Sleep(1 * time.Minute)
		// programar ETW?
	}
}
