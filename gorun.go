package main

import (
	"flag"
	"github.com/fsnotify/fsnotify"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	cmd       *exec.Cmd
	state     sync.Mutex
	appname   string
	eventTime = make(map[string]time.Time)
)

func Start() {
	if appname != "" {
		log.Println("Start", appname, "...")

		cmd = exec.Command("go", "run", appname)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		go cmd.Run()
	} else {
		err := Rebuild()

		if err != nil {
			return
		}

		log.Println("Start project ...")

		curpath, _ := os.Getwd()

		cmd = exec.Command("./" + filepath.Base(curpath))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		go cmd.Run()
	}
}

func Rebuild() error {
	state.Lock()
	defer state.Unlock()

	log.Println("Start rebuild project...")

	bcmd := exec.Command("go", "build")
	bcmd.Stdout = os.Stdout
	bcmd.Stderr = os.Stderr
	err := bcmd.Run()

	if err != nil {
		log.Println("============== Rebuild project failed ==============")

		return err
	}

	log.Println("Rebuild project success ...")

	return nil
}

func Stop() {
	defer func() {
		if e := recover(); e != nil {
			log.Println("Kill process error:", e)
		}
	}()
	if cmd != nil {
		log.Println("Kill running process:", cmd.Process.Pid)
		cmd.Process.Kill()
	}
}

func Restart() {
	Stop()
	Start()
}

func Watch() {
	path, _ := os.Getwd()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// walk dirs
	walkFn := func(path string, info os.FileInfo, err error) error {
		if strings.HasPrefix(filepath.Base(path), ".") {
			log.Println("Ignoring", path)

			return filepath.SkipDir
		}

		log.Println("Watching:", path)
		err = watcher.Add(path)
		if err != nil {
			log.Fatal(err)
		}

		return nil
	}

	if err := filepath.Walk(path, walkFn); err != nil {
		log.Println(err)
	}

	for {
		select {
		case event := <-watcher.Events:
			eventTime[event.Name] = time.Now()

			if event.Op == fsnotify.Write && strings.Contains(event.Name, ".go") {
				println("---------------------------------------------------")
				Restart()
			}

		case err := <-watcher.Errors:
			log.Println("error:", err)
		}
	}
}

func main() {
	flag.Parse()
	appname = flag.Arg(0)
	Start()

	Watch()
}
