package job

import "ingest-srv/pkg/cron"

func (h handler) Register() []cron.JobInfo {
	return []cron.JobInfo{
		{
			CronTime: h.cfg.HeartbeatCron,
			Handler:  h.DispatchDueTargets,
		},
	}
}
