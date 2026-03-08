package cron

import (
	"errors"

	robfigcron "github.com/robfig/cron/v3"
)

type HandleFunc func()

type Cron struct {
	cron        *robfigcron.Cron
	funcWrapper func(HandleFunc)
}

// Initialize returns a new CronJob
func New() Cron {
	parser := robfigcron.NewParser(
		robfigcron.SecondOptional |
			robfigcron.Minute |
			robfigcron.Hour |
			robfigcron.Dom |
			robfigcron.Month |
			robfigcron.Dow |
			robfigcron.Descriptor,
	)
	c := robfigcron.New(robfigcron.WithParser(parser))
	return Cron{
		cron: c,
	}
}

// SetFuncWrapper sets a function wrapper for CronJob
func (c *Cron) SetFuncWrapper(f func(HandleFunc)) {
	c.funcWrapper = f
}

func (c *Cron) getFuncWrapper() func(HandleFunc) {
	if c.funcWrapper == nil {
		return func(f HandleFunc) {
			f()
		}
	}
	return c.funcWrapper
}

// JobInfo is the job information
type JobInfo struct {
	CronTime string
	Handler  HandleFunc
}

// AddJob adds a new job to CronJob
func (c Cron) AddJob(info JobInfo) error {
	if info.CronTime == "" {
		return errors.New("invalid cron time")
	}

	fw := c.getFuncWrapper()

	_, err := c.cron.AddFunc(info.CronTime, func() {
		fw(info.Handler)
	})

	return err
}

// Start starts CronJob
func (c Cron) Start() {
	c.cron.Start()
}

// Stop stops CronJob
func (c Cron) Stop() {
	c.cron.Stop()
}
