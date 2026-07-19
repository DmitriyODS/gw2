package com.kodass.groovework;

import android.content.ContentResolver;
import android.content.Context;
import android.net.Uri;
import android.util.AttributeSet;
import android.util.Base64;
import android.view.inputmethod.EditorInfo;
import android.view.inputmethod.InputConnection;

import androidx.core.view.inputmethod.EditorInfoCompat;
import androidx.core.view.inputmethod.InputConnectionCompat;
import androidx.core.view.inputmethod.InputContentInfoCompat;

import com.getcapacitor.CapacitorWebView;

import java.io.ByteArrayOutputStream;
import java.io.InputStream;

// Наследник капаситоровского WebView, добавляющий приём КАРТИНОК С КЛАВИАТУРЫ
// (GBoard «вставить картинку»/GIF/стикер, буфер обмена с изображением). Такой
// контент приходит НЕ событием paste, а по rich-content API IME
// (InputConnection.commitContent) — обычный WebView его не принимает, потому что
// не объявляет поддерживаемые MIME. Здесь объявляем image/* и, получив картинку,
// отдаём её веб-слою (окно ловит 'gw:native-image' → вкладывает в активное поле
// сообщения, см. front/src/utils/nativeApp.js).
//
// Подключается подменой капаситоровского layout: наш
// res/layout/capacitor_bridge_layout_main.xml перекрывает библиотечный и ставит
// этот класс на тот же @id/webview, по которому Bridge находит WebView.
public class GrooveWebView extends CapacitorWebView {

    // Картинки с клавиатуры невелики (стикеры/GIF), но подстрахуемся от OOM —
    // тот же порог, что у серверного лимита вложений с небольшим запасом.
    private static final long MAX_IMAGE_BYTES = 26L * 1024 * 1024;

    private static final String[] ACCEPT_MIME = { "image/png", "image/jpeg", "image/gif", "image/webp" };

    public GrooveWebView(Context context, AttributeSet attrs) {
        super(context, attrs);
    }

    @Override
    public InputConnection onCreateInputConnection(EditorInfo outAttrs) {
        InputConnection ic = super.onCreateInputConnection(outAttrs);
        if (ic == null) return null;

        EditorInfoCompat.setContentMimeTypes(outAttrs, ACCEPT_MIME);

        InputConnectionCompat.OnCommitContentListener listener = (info, flags, opts) -> {
            if ((flags & InputConnectionCompat.INPUT_CONTENT_GRANT_READ_URI_PERMISSION) != 0) {
                try {
                    info.requestPermission();
                } catch (Exception e) {
                    return false;
                }
            }
            deliverToWeb(info);
            return true;
        };

        return InputConnectionCompat.createWrapper(ic, outAttrs, listener);
    }

    // Читает картинку в фоне (permission живёт до releasePermission), кодирует в
    // base64 и будит веб-слой на UI-потоке.
    private void deliverToWeb(final InputContentInfoCompat info) {
        new Thread(() -> {
            try {
                Uri uri = info.getContentUri();
                ContentResolver cr = getContext().getContentResolver();

                String mime = "image/png";
                if (info.getDescription() != null && info.getDescription().getMimeTypeCount() > 0) {
                    mime = info.getDescription().getMimeType(0);
                } else {
                    String t = cr.getType(uri);
                    if (t != null) mime = t;
                }

                byte[] bytes = readCapped(cr, uri);
                if (bytes == null) return;

                String b64 = Base64.encodeToString(bytes, Base64.NO_WRAP);
                String ext = mime.contains("/") ? mime.substring(mime.indexOf('/') + 1) : "png";
                String name = "image-" + System.currentTimeMillis() + "." + ext;

                final String js = "window.dispatchEvent(new CustomEvent('gw:native-image',{detail:{"
                    + "data:" + jsStr(b64) + ",mime:" + jsStr(mime) + ",name:" + jsStr(name) + "}}));";
                post(() -> evaluateJavascript(js, null));
            } catch (Exception ignored) {
            } finally {
                try {
                    info.releasePermission();
                } catch (Exception ignored) {
                }
            }
        }).start();
    }

    private static byte[] readCapped(ContentResolver cr, Uri uri) throws Exception {
        try (InputStream in = cr.openInputStream(uri)) {
            if (in == null) return null;
            ByteArrayOutputStream bos = new ByteArrayOutputStream();
            byte[] buf = new byte[8192];
            int n;
            long total = 0;
            while ((n = in.read(buf)) != -1) {
                total += n;
                if (total > MAX_IMAGE_BYTES) return null;
                bos.write(buf, 0, n);
            }
            return bos.toByteArray();
        }
    }

    // Безопасный JS-строковый литерал (экранируем кавычки/слэши/переводы строк).
    private static String jsStr(String s) {
        StringBuilder sb = new StringBuilder("\"");
        for (int i = 0; i < s.length(); i++) {
            char c = s.charAt(i);
            switch (c) {
                case '\\': sb.append("\\\\"); break;
                case '"': sb.append("\\\""); break;
                case '\n': sb.append("\\n"); break;
                case '\r': sb.append("\\r"); break;
                default: sb.append(c);
            }
        }
        return sb.append('"').toString();
    }
}
