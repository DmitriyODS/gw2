package com.kodass.groovework;

import android.app.Notification;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.Context;
import android.content.Intent;
import android.service.notification.StatusBarNotification;

import androidx.core.app.NotificationCompat;
import androidx.core.app.NotificationManagerCompat;
import androidx.core.app.Person;
import androidx.core.app.RemoteInput;

// Уведомления о сообщениях чата (Android-обёртка строит их сама из data-only
// FCM-пуша). Каждый диалог — своя переписка в стиле MessagingStyle: входящие
// от одного собеседника накапливаются в ОДНОМ уведомлении (группировка по
// отправителю), несколько диалогов сворачиваются в общую пачку. У каждого
// уведомления — действие «Ответить» с полем прямо в шторке (RemoteInput),
// отправку выполняет MessageReplyReceiver.
final class MessageNotifications {

    static final String CHANNEL = "messages";
    static final String GROUP = "gw_messages";
    static final String KEY_REPLY = "gw_reply_text";
    static final String EXTRA_CONV_ID = "conv_id";
    static final String EXTRA_SENDER = "sender_name";
    private static final int SUMMARY_ID = 42100;

    private MessageNotifications() {}

    // Стабильный id уведомления на диалог — повторный пуш обновляет ту же
    // переписку, а не плодит карточки.
    static int notifId(String convId) {
        return (GROUP + ':' + convId).hashCode() & 0x7fffffff;
    }

    // Входящее сообщение: докидываем к существующей переписке диалога либо
    // заводим новую.
    static void incoming(Context ctx, String convId, String senderName, String text) {
        NotificationManager nm = ctx.getSystemService(NotificationManager.class);
        if (nm == null || !NotificationManagerCompat.from(ctx).areNotificationsEnabled()) return;
        NotificationCompat.MessagingStyle style = existingStyle(nm, notifId(convId), senderName);
        style.addMessage(text, System.currentTimeMillis(), new Person.Builder().setName(senderName).build());
        post(ctx, nm, convId, senderName, style);
    }

    // Успешно отправленный из шторки ответ — показываем его в переписке как
    // своё сообщение (null-отправитель у MessagingStyle означает «я»).
    static void appendSelfReply(Context ctx, String convId, String senderName, String text) {
        NotificationManager nm = ctx.getSystemService(NotificationManager.class);
        if (nm == null) return;
        NotificationCompat.MessagingStyle style = existingStyle(nm, notifId(convId), senderName);
        style.addMessage(text, System.currentTimeMillis(), (Person) null);
        post(ctx, nm, convId, senderName, style);
    }

    // Переотрисовать уведомление как есть — гасит «крутилку» после неудачной
    // отправки ответа (система крутит спиннер, пока уведомление не обновят).
    static void repost(Context ctx, String convId, String senderName) {
        NotificationManager nm = ctx.getSystemService(NotificationManager.class);
        if (nm == null) return;
        post(ctx, nm, convId, senderName, existingStyle(nm, notifId(convId), senderName));
    }

    private static NotificationCompat.MessagingStyle existingStyle(NotificationManager nm, int id, String senderName) {
        try {
            for (StatusBarNotification sbn : nm.getActiveNotifications()) {
                if (sbn.getId() == id) {
                    NotificationCompat.MessagingStyle s =
                        NotificationCompat.MessagingStyle.extractMessagingStyleFromNotification(sbn.getNotification());
                    if (s != null) return s;
                }
            }
        } catch (Exception ignored) {
        }
        NotificationCompat.MessagingStyle s =
            new NotificationCompat.MessagingStyle(new Person.Builder().setName("Вы").build());
        s.setConversationTitle(senderName);
        return s;
    }

    private static void post(Context ctx, NotificationManager nm, String convId, String senderName,
                             NotificationCompat.MessagingStyle style) {
        int id = notifId(convId);

        RemoteInput remoteInput = new RemoteInput.Builder(KEY_REPLY)
            .setLabel("Сообщение…")
            .build();
        Intent replyIntent = new Intent(ctx, MessageReplyReceiver.class)
            .putExtra(EXTRA_CONV_ID, convId)
            .putExtra(EXTRA_SENDER, senderName);
        // FLAG_MUTABLE обязателен — систему кладёт введённый текст в этот intent.
        PendingIntent replyPI = PendingIntent.getBroadcast(ctx, id, replyIntent,
            PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_MUTABLE);
        NotificationCompat.Action reply = new NotificationCompat.Action.Builder(
            R.drawable.ic_launcher_foreground, "Ответить", replyPI)
            .addRemoteInput(remoteInput)
            .setAllowGeneratedReplies(false)
            .setSemanticAction(NotificationCompat.Action.SEMANTIC_ACTION_REPLY)
            .build();

        Intent open = new Intent(ctx, MainActivity.class)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_SINGLE_TOP);
        PendingIntent tap = PendingIntent.getActivity(ctx, id, open,
            PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);

        Notification n = new NotificationCompat.Builder(ctx, CHANNEL)
            .setSmallIcon(R.drawable.ic_launcher_foreground)
            .setStyle(style)
            .setContentIntent(tap)
            .setAutoCancel(true)
            .addAction(reply)
            .setGroup(GROUP)
            .setCategory(NotificationCompat.CATEGORY_MESSAGE)
            .build();
        nm.notify(id, n);

        // Сводка группы — несколько диалогов сворачиваются в одну шапку.
        Notification summary = new NotificationCompat.Builder(ctx, CHANNEL)
            .setSmallIcon(R.drawable.ic_launcher_foreground)
            .setGroup(GROUP)
            .setGroupSummary(true)
            .setAutoCancel(true)
            .build();
        nm.notify(SUMMARY_ID, summary);
    }
}
