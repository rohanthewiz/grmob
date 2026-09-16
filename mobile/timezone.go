package mobile

import "time"

// SetTimeZone makes the device's time zone Go's local one. The shell calls it
// once, beside SetDataDir, before anything renders.
//
// # Why the shell has to say
//
// Go finds time.Local from $TZ or /etc/localtime. An Android app process has
// neither, so without this every time.Now() in a bound app is UTC: lesson
// 4.19's clocks read five hours out on a Chicago device, and an alarm set for
// "the next minute" was labelled 4:47 PM on a phone showing 11:47. The
// absolute instants were right — only the wall clock was wrong — which is why
// the alarm still fired on time. iOS has /etc/localtime, but the shell passes
// the zone there too so both hosts follow one rule and a missing file cannot
// silently put an app in UTC.
//
// name is the IANA id the platform reports (java.util.TimeZone.getDefault().id,
// TimeZone.current.identifier). Go's time package reads Android's system tzdata
// and iOS's zoneinfo directly, so LoadLocation normally succeeds and daylight
// saving transitions are right. If it fails, offsetSeconds (the zone's current
// offset from UTC) gives a fixed zone: wrong across the next DST change, but
// right today, which is better than UTC.
//
// # Once, before rendering
//
// time.Local is a plain variable that every goroutine reads unsynchronised,
// so writing it while timers and renders run is a data race. It is set before
// RenderInitial and not again: a zone change while the app runs (travel, the
// user editing Settings) is picked up on the next launch.
func SetTimeZone(name string, offsetSeconds int) {
	time.Local = resolveTimeZone(name, offsetSeconds)
}

// resolveTimeZone is SetTimeZone's choice of location, separate so it can be
// tested without writing time.Local.
func resolveTimeZone(name string, offsetSeconds int) *time.Location {
	if name != "" {
		if loc, err := time.LoadLocation(name); err == nil {
			return loc
		}
	}
	if name == "" {
		name = "Local"
	}
	return time.FixedZone(name, offsetSeconds)
}
