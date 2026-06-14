package com.kodass.groovework.ui.calls

import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import io.livekit.android.compose.ui.ScaleType
import io.livekit.android.compose.ui.VideoTrackView
import io.livekit.android.room.Room
import io.livekit.android.room.track.VideoTrack

// Надёжный рендер видеотрека LiveKit поверх официального компонента
// livekit-android-compose-components (он сам ведёт visibility/lifecycle sink —
// это и чинит «чёрное» видео ручного TextureViewRenderer).
@Composable
fun CallVideo(
    room: Room,
    track: VideoTrack?,
    modifier: Modifier = Modifier,
    mirror: Boolean = false,
    fit: Boolean = false,
) {
    VideoTrackView(
        videoTrack = track,
        passedRoom = room,
        mirror = mirror,
        scaleType = if (fit) ScaleType.FitInside else ScaleType.Fill,
        modifier = modifier,
    )
}
