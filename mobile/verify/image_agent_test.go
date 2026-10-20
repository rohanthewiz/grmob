package verify

import (
	"strings"
	"testing"
)

// The Android shell's images are fetched with a descriptive User-Agent.
//
// Coil 2 fetches core.Image sources through OkHttp, whose default agent is
// "okhttp/4.x", and Wikimedia refuses that with a 403 ("Please set a
// user-agent and respect our robot policy"). The browser and iOS send
// descriptive agents of their own, so a Commons photo loaded on two hosts
// and drew as an empty box on Android, with nothing in logcat. Seen on a
// Galaxy Z Fold6; GrMobApplication.kt is the fix, and these pins hold the
// three links it needs:
//
//	manifest      names GrMobApplication, or Android never instantiates it
//	factory       GrMobApplication is Coil's ImageLoaderFactory, which is how
//	              the singleton loader every AsyncImage uses is configured
//	interceptor   sets User-Agent only when the request has none, so a
//	              caller's own agent is kept
//
// Nothing here runs OkHttp; the device is where the photo was seen to load.
func TestTheAndroidShellSendsADescriptiveImageUserAgent(t *testing.T) {
	manifest := nativeFile("android", "app", "src", "main", "AndroidManifest.xml")
	if !strings.Contains(valuesIn(t, manifest), `android:name="com.grmob.app.GrMobApplication"`) {
		t.Errorf("%s does not name com.grmob.app.GrMobApplication as the application; "+
			"Coil falls back to OkHttp's default User-Agent and Wikimedia images 403", manifest)
	}

	app := nativeFile("android", "app", "src", "main", "java", "com", "grmob", "app", "GrMobApplication.kt")
	for _, c := range []struct{ decl, expr, why string }{
		{"class GrMobApplication", "Application(), ImageLoaderFactory",
			"Coil configures its singleton loader through the Application's ImageLoaderFactory"},
		{"override fun newImageLoader()", ".okHttpClient {",
			"the agent is set on the OkHttp client Coil fetches with"},
		{"override fun newImageLoader()", `request.header("User-Agent") != null`,
			"a request that already carries an agent keeps it"},
		{"override fun newImageLoader()", `.header("User-Agent", agent)`,
			"every other request gets the descriptive agent"},
	} {
		if !strings.Contains(valuesOf(t, app, c.decl), c.expr) {
			t.Errorf("%s: %s has no %q — %s", app, c.decl, c.expr, c.why)
		}
	}
}
