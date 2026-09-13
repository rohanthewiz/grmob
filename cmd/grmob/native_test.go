package main

import "testing"

// TestCountBootedReadsSimctlListing holds the -run preflight to simctl's JSON
// shape, using listings taken from `xcrun simctl list devices booted -j` (Xcode
// with the iOS 26.5 runtime) with and without a booted iPhone 17 Pro, plus the
// unfiltered case countBooted guards against.
func TestCountBootedReadsSimctlListing(t *testing.T) {
	cases := []struct {
		name    string
		listing string
		want    int
	}{{
		name: "none booted: the runtime group is present and empty",
		listing: `{"devices" : {"com.apple.CoreSimulator.SimRuntime.iOS-26-5" : [

    ]}}`,
		want: 0,
	}, {
		name: "one booted",
		listing: `{"devices": {"com.apple.CoreSimulator.SimRuntime.iOS-26-5": [
			{"name": "iPhone 17 Pro", "udid": "7572B953-1590-419D-A7B7-D06B4CB60E32", "state": "Booted", "isAvailable": true}
		]}}`,
		want: 1,
	}, {
		name: "an unfiltered listing counts only the booted device",
		listing: `{"devices": {
			"com.apple.CoreSimulator.SimRuntime.iOS-26-5": [
				{"name": "iPhone 17 Pro", "state": "Booted"},
				{"name": "iPhone 17", "state": "Shutdown"}
			],
			"com.apple.CoreSimulator.SimRuntime.watchOS-26-0": [
				{"name": "Apple Watch", "state": "Shutdown"}
			]
		}}`,
		want: 1,
	}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := countBooted([]byte(c.listing))
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("countBooted = %d, want %d", got, c.want)
			}
		})
	}

	if _, err := countBooted([]byte("-- iOS 26.5 --")); err == nil {
		t.Error("countBooted accepted simctl's text table; want an error so the preflight stands aside")
	}
}
