package com.kodass.groovework.ui.common

import androidx.compose.material3.LocalTextStyle
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.LinkAnnotation
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.TextLinkStyles
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.withLink

private val URL_REGEX = Regex(
    "(https?://[^\\s]+|www\\.[^\\s]+)",
    RegexOption.IGNORE_CASE,
)

// Пунктуация, прилипающая к концу ссылки в обычной речи — её не включаем в URL.
private val TRAILING_PUNCTUATION = charArrayOf('.', ',', ')', ']', '}', '!', '?', ';', ':', '»', '"', '\'')

/**
 * Текст, в котором URL'ы становятся кликабельными (открываются в браузере через
 * стандартный LocalUriHandler). Используется для пользовательского контента
 * (комментарии к задачам и т.п.), где ссылки приходят сырой строкой.
 */
@Composable
fun LinkifiedText(
    text: String,
    modifier: Modifier = Modifier,
    style: TextStyle = LocalTextStyle.current,
) {
    val linkColor = MaterialTheme.colorScheme.primary
    val annotated = remember(text, linkColor) { buildLinkifiedString(text, linkColor) }
    Text(text = annotated, modifier = modifier, style = style)
}

private fun buildLinkifiedString(text: String, linkColor: Color): AnnotatedString = buildAnnotatedString {
    val linkStyles = TextLinkStyles(
        style = SpanStyle(color = linkColor, textDecoration = TextDecoration.Underline),
    )
    var last = 0
    for (match in URL_REGEX.findAll(text)) {
        val start = match.range.first
        if (start > last) append(text.substring(last, start))

        val raw = match.value
        val trimmed = raw.trimEnd(*TRAILING_PUNCTUATION)
        val url = if (trimmed.startsWith("www.", ignoreCase = true)) "https://$trimmed" else trimmed
        withLink(LinkAnnotation.Url(url, linkStyles)) { append(trimmed) }
        if (trimmed.length < raw.length) append(raw.substring(trimmed.length))

        last = match.range.last + 1
    }
    if (last < text.length) append(text.substring(last))
}
