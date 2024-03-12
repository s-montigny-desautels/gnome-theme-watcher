package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	isWatcher bool
	isServer  bool
	once      bool
)

func init() {
	flag.BoolVar(&isWatcher, "watch", false, "Watch the user theme and output it to stdout")
	flag.BoolVar(&once, "once", false, "Watch only once and exit on first change")
	flag.BoolVar(&isServer, "server", false, "Watch the user theme and run configured scripts")
}

func queryTheme() (isDark bool) {
	args := []string{
		"call", "--session",
		"--dest=org.freedesktop.portal.Desktop",
		"--object-path", "/org/freedesktop/portal/desktop",
		"--method", "org.freedesktop.portal.Settings.Read",
		"org.freedesktop.appearance", "color-scheme",
	}

	out, err := exec.Command("gdbus", args...).Output()
	if err != nil {
		log.Fatal(err)
	}

	return strings.Contains(string(out), "<<uint32 1>>")
}

func watchTheme(cb func(bool), once bool) {
	args := []string{
		"monitor",
		"--session",
		"--dest=org.freedesktop.portal.Desktop",
		"--object-path", "/org/freedesktop/portal/desktop",
	}
	for {
		cmd := exec.Command("gdbus", args...)

		cmdReader, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}

		scanner := bufio.NewScanner(cmdReader)
		go func() {
			var lastVal bool
			firstRun := true

			for scanner.Scan() {
				line := scanner.Text()

				if strings.Contains(line, "org.gnome.desktop.interface") && strings.Contains(line, "color-scheme") {
					isDark := strings.Contains(line, "<'prefer-dark'>")

					if isDark != lastVal || firstRun {
						firstRun = false
						lastVal = isDark

						cb(isDark)

						if once {
							cmd.Process.Kill()
						}
					}
				}
			}
		}()

		err = cmd.Start()
		if err != nil {
			log.Fatal(err)
		}

		cmd.Wait()
		if once {
			return
		}

		time.Sleep(1 * time.Second)
	}
}

func runScripts(isDark bool) {
	localConfig := filepath.Join(os.Getenv("HOME"), ".config", "gnome-theme-watcher")
	files, err := os.ReadDir(filepath.Join(localConfig, "scripts"))
	if err != nil {
		fmt.Printf("failed to read scripts dir: %s", err)
		return
	}

	arg := "0"
	if isDark {
		arg = "1"
	}

	for _, file := range files {
		_, err := exec.Command(filepath.Join(localConfig, "scripts", file.Name()), arg).Output()
		if err != nil {
			fmt.Printf("failed to exec scripts %s: %s", file, err)
		}
	}
}

func logIsDark(isDark bool) {
	var out int
	if isDark {
		out = 1
	}

	fmt.Println(out)
}

func main() {
	flag.Parse()

	switch {
	case isWatcher:
		watchTheme(logIsDark, once)
	case isServer:
		watchTheme(runScripts, once)
	default:
		logIsDark(queryTheme())
	}
}
