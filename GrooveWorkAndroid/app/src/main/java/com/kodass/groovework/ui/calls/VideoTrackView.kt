package com.kodass.groovework.ui.calls

import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.viewinterop.AndroidView
import io.livekit.android.renderer.TextureViewRenderer
import io.livekit.android.room.Room
import io.livekit.android.room.track.VideoTrack

// Рендер видеотрека LiveKit; смена трека перевешивает sink без пересоздания вью.
@Composable
fun VideoTrackView(room: Room, track: VideoTrack?, modifier: Modifier = Modifier) {
    AndroidView(
        factory = { context ->
            TextureViewRenderer(context).also { renderer ->
                room.initVideoRenderer(renderer)
            }
        },
        update = { renderer ->
            val previous = renderer.tag as? VideoTrack
            if (previous !== track) {
                previous?.removeRenderer(renderer)
                track?.addRenderer(renderer)
                renderer.tag = track
            }
        },
        onRelease = { renderer ->
            (renderer.tag as? VideoTrack)?.removeRenderer(renderer)
            renderer.release()
        },
        modifier = modifier,
    )
}
