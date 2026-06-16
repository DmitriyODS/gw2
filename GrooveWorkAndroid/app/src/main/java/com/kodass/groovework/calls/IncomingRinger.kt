package com.kodass.groovework.calls

import android.content.Context
import android.media.AudioAttributes
import android.media.MediaPlayer
import android.media.RingtoneManager
import android.os.VibrationAttributes
import android.os.VibrationEffect
import android.os.Vibrator
import android.os.VibratorManager

// Звонилка входящего: зацикленный системный рингтон + зацикленная вибрация (как
// настоящий звонок). Живёт в процессе, удерживаемом foreground-сервисом звонка.
// Останавливается на ответ/отклон/завершение.
class IncomingRinger(private val context: Context) {
    private var player: MediaPlayer? = null
    private var vibrator: Vibrator? = null

    fun start() {
        stop()
        startRingtone()
        startVibration()
    }

    fun stop() {
        runCatching {
            player?.stop()
            player?.release()
        }
        player = null
        runCatching { vibrator?.cancel() }
        vibrator = null
    }

    private fun startRingtone() {
        runCatching {
            val uri = RingtoneManager.getDefaultUri(RingtoneManager.TYPE_RINGTONE) ?: return
            player = MediaPlayer().apply {
                setDataSource(context, uri)
                setAudioAttributes(
                    AudioAttributes.Builder()
                        .setUsage(AudioAttributes.USAGE_NOTIFICATION_RINGTONE)
                        .setContentType(AudioAttributes.CONTENT_TYPE_SONIFICATION)
                        .build()
                )
                isLooping = true
                prepare()
                start()
            }
        }
    }

    private fun startVibration() {
        runCatching {
            val vib = (context.getSystemService(Context.VIBRATOR_MANAGER_SERVICE) as VibratorManager)
                .defaultVibrator
            vibrator = vib
            // Пауза-вибро-пауза, зациклено (repeat = 0 — с начала массива).
            val pattern = longArrayOf(0, 800, 600)
            val effect = VibrationEffect.createWaveform(pattern, 0)
            val attrs = VibrationAttributes.Builder()
                .setUsage(VibrationAttributes.USAGE_RINGTONE)
                .build()
            vib.vibrate(effect, attrs)
        }
    }
}
