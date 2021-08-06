package daemons

import (
	"time"

	"github.com/zsmartex/finex/jobs"
	"github.com/zsmartex/finex/jobs/cron"
)

type CronJob struct {
	Running bool
	Jobs    []jobs.Job
}

func NewCronJob() *CronJob {
	jobs := []jobs.Job{&cron.GlobalPriceJob{}}

	return &CronJob{Running: true, Jobs: jobs}
}

func (c *CronJob) Stop() {
	c.Running = false
}

func (c *CronJob) Start() {
	for _, job := range c.Jobs {
		go c.Process(job)
	}

	for {
		// Empty for to make it running for ever
		time.Sleep(1 * time.Second)
	}
}

func (c *CronJob) Process(job jobs.Job) {
	for {
		if !c.Running {
			break
		}

		job.Process()
	}
}
