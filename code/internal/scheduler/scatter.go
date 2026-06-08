package scheduler

import (
	"hash/fnv"
	"strconv"
	"time"
)

func Scatter(taskID string, triggerUnix int64) time.Duration {
	h := fnv.New32a()
	h.Write([]byte(taskID + "|" + strconv.FormatInt(triggerUnix, 10)))
	ms := int(h.Sum32() % 1000)
	return time.Duration(ms) * time.Millisecond
}
