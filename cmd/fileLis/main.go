package main

import (
	"github.com/fsnotify/fsnotify"
	"log"
)

func main() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("NewWatcher failed: ", err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		defer close(done)

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event.Name, event.Op)
				// Writing in this way reduces some messages:
				if event.Op&fsnotify.Rename == fsnotify.Rename {
					log.Printf("%s %s\n", event.Name, event.Op)
				} else if event.Op&fsnotify.Create == fsnotify.Create {
					log.Printf("%s %s\n", event.Name, event.Op)
				} else if event.Op&fsnotify.Write == fsnotify.Write {
					log.Printf("%s %s\n", event.Name, event.Op)
				} else if event.Op&fsnotify.Remove == fsnotify.Remove {
					log.Printf("%s %s\n", event.Name, event.Op)
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add("./")
	if err != nil {
		log.Fatal("Add failed:", err)
	}
	<-done
}
