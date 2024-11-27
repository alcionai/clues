package clog

type labels struct {
	// good for categorizing debugging-level api call info.
	APIResponse,
	// when you want your log to cause a lot of noise.
	AlarmWhenYouSeeThis,
	// for things that happen while releasing resources.
	Cleanup,
	// for showcasing the runtime configuration of your app
	ConfigurationOverview,
	// everything that you want to know about the process
	// at the time of its conclusion.
	EndOfRun,
	// maybe you have more error logs than you have failure causes.
	// That's okay, it's what searching by labes is here for.
	Failures,
	// when you want debug logging to include info about every
	// little thing  that gets handled through the process.  No,
	// honestly, we don't expect you to use this label.  It's just
	// an example for fun.
	EveryLittleThing,
	// when debugging the progress of a process and you want to
	// include logs that track the completion of long running
	// processes.
	ProgressTracker,
	// everything that you want to know about the state of the
	// application when you kick off a new process.
	StartOfRun,
	// who needs a logging level when you can use a label instead?
	Warning string
}

// Labels provides a example set of labels for use with
// clog.Ctx(ctx).Label(). This list is not canonical, or even
// important to clog.  It's just here to help give you some ideas.
func Labels() labels {
	return labels{
		APIResponse:           "api_response",
		AlarmWhenYouSeeThis:   "alarm_when_you_see_this",
		Cleanup:               "cleanup",
		ConfigurationOverview: "configuration_overview",
		EndOfRun:              "end_of_run",
		Failures:              "failures",
		EveryLittleThing:      "every_little_thing",
		ProgressTracker:       "progress_tracker",
		StartOfRun:            "start_of_run",
		Warning:               "warning",
	}
}
