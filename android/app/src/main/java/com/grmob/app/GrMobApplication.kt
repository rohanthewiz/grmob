package com.grmob.app

import android.app.Application
import android.os.Build
import coil.ImageLoader
import coil.ImageLoaderFactory
import okhttp3.OkHttpClient

// GrMobApplication gives Coil, which draws every core.Image, an ImageLoader
// that sends a descriptive User-Agent.
//
// # Why
//
// Coil 2 fetches through OkHttp, whose default User-Agent is "okhttp/4.x".
// Image hosts that enforce a robot policy refuse that outright. Wikimedia
// answers 403 "Please set a user-agent and respect our robot policy" (see
// https://w.wiki/4wJS), so a Commons photo in a core.Image drew as an empty
// box on Android while the browser and iOS, which send descriptive agents
// of their own, loaded them fine. (Found in an app on a Galaxy Z Fold6.)
// It is the same lesson GrMobMapView learned for OSM's tile servers, which
// is why createOsmMap sets osmdroid's userAgentValue; Coil was never given
// the equivalent.
//
// # How
//
// Coil asks the Application for its singleton loader when the Application
// implements ImageLoaderFactory, so this is the one place to configure it.
// AsyncImage calls in the runtime need no change. The agent is set by an
// application interceptor, which runs before OkHttp's BridgeInterceptor
// fills in the default, and only when a request has no User-Agent yet, so
// a caller that sets its own keeps it.
//
// The shape follows Wikimedia's policy: client name/version, then a way to
// identify the app (its application ID), then the platform:
//
//	My App/1.0 (com.example.myapp; Android 14) grmob
//
// Everything else stays at Coil's builder defaults, so images behave as
// before apart from actually arriving.
class GrMobApplication : Application(), ImageLoaderFactory {
    override fun newImageLoader(): ImageLoader {
        val agent = userAgent()
        return ImageLoader.Builder(this)
            .okHttpClient {
                OkHttpClient.Builder()
                    .addInterceptor { chain ->
                        val request = chain.request()
                        if (request.header("User-Agent") != null) {
                            chain.proceed(request)
                        } else {
                            chain.proceed(request.newBuilder().header("User-Agent", agent).build())
                        }
                    }
                    .build()
            }
            .build()
    }

    // userAgent builds "Label/version (applicationId; Android release) grmob"
    // from the installed package, so it follows grmob.json's name and ID and
    // the build's versionName without restating any of them here.
    private fun userAgent(): String {
        val label = applicationInfo.loadLabel(packageManager).toString()
        val version = try {
            @Suppress("DEPRECATION")
            packageManager.getPackageInfo(packageName, 0).versionName ?: "0"
        } catch (e: Exception) {
            "0"
        }
        return "$label/$version ($packageName; Android ${Build.VERSION.RELEASE}) grmob"
    }
}
