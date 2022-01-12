package log

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type LEVEL byte

const (
	DEBUG LEVEL = iota
	INFO
	WARN
	ERROR
)

const DATE_FORMAT = "2006-01-02"

type FileLogger struct {
	fileDir       string
	fileName      string
	prefix        string
	logLevel      LEVEL
	logFile       *os.File
	date          *time.Time
	lg            *log.Logger
	mu            *sync.RWMutex
	logChan       chan string
	stopTikerChan chan bool
}

var fileLogger *FileLogger

func Init(fileDir, fileName, prefix, level string) error {
	CloseLogger()

	f := &FileLogger{
		fileDir:       fileDir,
		fileName:      fileName,
		prefix:        prefix,
		mu:            new(sync.RWMutex),
		logChan:       make(chan string, 5000),
		stopTikerChan: make(chan bool, 1),
	}

	switch strings.ToUpper(level) {
	case "DEBUG":
		f.logLevel = DEBUG
	case "WARN":
		f.logLevel = WARN
	case "ERROR":
		f.logLevel = ERROR
	default:
		f.logLevel = INFO
	}

	t, _ := time.Parse(DATE_FORMAT, time.Now().Format(DATE_FORMAT))
	f.date = &t

	f.isExistOrCreateFileDir()

	fullFileName := filepath.Join(f.fileDir, f.fileName+".log")
	file, err := os.OpenFile(fullFileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	f.logFile = file

	f.lg = log.New(f.logFile, prefix, log.LstdFlags|log.Lmicroseconds)

	go f.logWriter()
	go f.fileMonitor()

	fileLogger = f

	return nil
}

func (f *FileLogger) isExistOrCreateFileDir() {
	_, err := os.Stat(f.fileDir)
	if err != nil && !os.IsExist(err) {
		os.Mkdir(f.fileDir, 0755)
	}
}

func (f *FileLogger) logWriter() {
	defer func() { recover() }()

	for {
		str, ok := <-f.logChan
		if !ok {
			return
		}

		f.mu.RLock()
		f.lg.Output(2, str)
		f.mu.RUnlock()
	}
}

func (f *FileLogger) fileMonitor() {
	defer func() { recover() }()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if f.isMustSplit() {
				if err := f.split(); err != nil {
					Error("Log split error: %v\n", err)
				}
			}
		case <-f.stopTikerChan:
			return
		}
	}
}

func (f *FileLogger) isMustSplit() bool {
	t, _ := time.Parse(DATE_FORMAT, time.Now().Format(DATE_FORMAT))
	return t.After(*f.date)
}

func (f *FileLogger) split() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	logFile := filepath.Join(f.fileDir, f.fileName)
	logFileBak := logFile + "-" + f.date.Format(DATE_FORMAT) + ".log"

	if f.logFile != nil {
		f.logFile.Close()
	}

	err := os.Rename(logFile, logFileBak)
	if err != nil {
		return err
	}

	t, _ := time.Parse(DATE_FORMAT, time.Now().Format(DATE_FORMAT))
	f.date = &t

	f.logFile, err = os.OpenFile(logFile, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return err
	}

	f.lg = log.New(f.logFile, f.prefix, log.LstdFlags|log.Lmicroseconds)

	return nil
}

func CloseLogger() {
	if fileLogger != nil {
		fileLogger.stopTikerChan <- true
		close(fileLogger.stopTikerChan)
		close(fileLogger.logChan)
		fileLogger.lg = nil
		fileLogger.logFile.Close()
	}
}

func Printf(format string, v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	fileLogger.logChan <- fmt.Sprintf("[%v:%v]", filepath.Base(file), line) + fmt.Sprintf(format, v...)
}

func Print(v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	fileLogger.logChan <- fmt.Sprintf("[%v:%v]", filepath.Base(file), line) + fmt.Sprint(v...)
}

func Println(v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	fileLogger.logChan <- fmt.Sprintf("[%v:%v]", filepath.Base(file), line) + fmt.Sprintln(v...)
}

func Debug(format string, v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	if fileLogger.logLevel <= DEBUG {
		fileLogger.logChan <- fmt.Sprintf("[%v:%v]", filepath.Base(file), line) + fmt.Sprintf("[DEBUG]"+format, v...)
	}
}

func Info(format string, v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	if fileLogger.logLevel <= INFO {
		fileLogger.logChan <- fmt.Sprintf("[%v:%v]", filepath.Base(file), line) + fmt.Sprintf("[INFO]"+format, v...)
	}
}

func Warn(format string, v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	if fileLogger.logLevel <= WARN {
		fileLogger.logChan <- fmt.Sprintf("[%v:%v]", filepath.Base(file), line) + fmt.Sprintf("[WARN]"+format, v...)
	}
}

func Error(format string, v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	if fileLogger.logLevel <= ERROR {
		fileLogger.logChan <- fmt.Sprintf("[%v:%v]", filepath.Base(file), line) + fmt.Sprintf("[ERROR]"+format, v...)
	}
}
