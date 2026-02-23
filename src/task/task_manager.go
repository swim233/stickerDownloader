package task

import (
	"context"

	"github.com/swim233/StickerDownloader/lib"
)

func TaskManager(taskChan <-chan lib.Task) {
	for {
		task := <-taskChan
		go TaskHandler(task, context.Background())
        
	}
}
