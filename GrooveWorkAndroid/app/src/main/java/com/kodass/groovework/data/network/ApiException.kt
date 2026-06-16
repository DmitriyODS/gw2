package com.kodass.groovework.data.network

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.jsonObject
import retrofit2.HttpException
import java.io.IOException

// Единый формат ошибок REST: {"error": CODE, "message": строка | объект-валидации} + HTTP-статус.
class ApiException(
    val code: String,
    override val message: String,
    val status: Int,
    val retryAfterSec: Int? = null,
    // Extra-поле email: приходит при EMAIL_NOT_VERIFIED — ведём на экран кода.
    val email: String? = null,
) : Exception(message)

fun parseApiError(json: Json, status: Int, body: String?): ApiException {
    if (body.isNullOrBlank()) return ApiException("HTTP_$status", "Ошибка сервера ($status)", status)
    return try {
        val obj = json.parseToJsonElement(body).jsonObject
        val code = (obj["error"] as? JsonPrimitive)?.content ?: "HTTP_$status"
        val message = when (val msg = obj["message"]) {
            is JsonPrimitive -> msg.content
            is JsonObject -> msg.entries.firstNotNullOfOrNull { (_, v) ->
                when (v) {
                    is JsonArray -> (v.firstOrNull() as? JsonPrimitive)?.content
                    is JsonPrimitive -> v.content
                    else -> null
                }
            } ?: "Ошибка валидации"
            else -> "Ошибка сервера ($status)"
        }
        val retryAfter = (obj["retry_after_sec"] as? JsonPrimitive)?.content?.toIntOrNull()
        val email = (obj["email"] as? JsonPrimitive)?.content
        ApiException(code, message, status, retryAfter, email)
    } catch (_: Exception) {
        ApiException("HTTP_$status", "Ошибка сервера ($status)", status)
    }
}

// Обёртка вызовов Retrofit: HttpException/IOException → ApiException с русским текстом.
suspend fun <T> apiCall(json: Json, block: suspend () -> T): T {
    try {
        return block()
    } catch (e: ApiException) {
        throw e
    } catch (e: HttpException) {
        throw parseApiError(json, e.code(), e.response()?.errorBody()?.string())
    } catch (e: IOException) {
        throw ApiException("NETWORK_ERROR", "Сервер недоступен", 0)
    }
}
