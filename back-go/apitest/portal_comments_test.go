package apitest

// API-тесты обсуждений портала: дерево ответов на комментарии (произвольная
// вложенность) и лайки-toggle.

import (
	"fmt"
	"net/http"
	"testing"
)

// findComment — комментарий по id в ответе GET /posts/:id/comments.
func findComment(t *testing.T, list []any, id int64) map[string]any {
	t.Helper()
	for _, item := range list {
		c, ok := item.(map[string]any)
		if ok && int64(c["id"].(float64)) == id {
			return c
		}
	}
	t.Fatalf("комментарий %d не найден в обсуждении", id)
	return nil
}

// TestPortalCommentThread — ответы образуют дерево: ответ на ответ хранит
// своего родителя, ответить на комментарий чужого поста нельзя, а удаление
// ветки уносит её целиком (каскад FK).
func TestPortalCommentThread(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Портал-треды "))
	author := newMember(t, admin, companyID, roleEmployee)
	mate := newMember(t, admin, companyID, roleEmployee)

	postID := createPost(t, author, "Обсудим?")
	otherPostID := createPost(t, author, "Другой пост")

	// Корневой комментарий.
	r := portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/posts/%d/comments", postID),
		mate.Token, map[string]any{"text": "Отличный пост!"})
	requireStatus(t, r, 201, "корневой комментарий")
	rootID := int64(r.Num("id"))
	if r.JSON["reply_to_id"] != nil {
		t.Fatalf("корневой комментарий не должен иметь родителя: %s", r.Raw)
	}

	// Ответ на него.
	r = portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/posts/%d/comments", postID),
		author.Token, map[string]any{"text": "Согласен", "reply_to_id": rootID})
	requireStatus(t, r, 201, "ответ на комментарий")
	replyID := int64(r.Num("id"))
	if r.JSON["reply_to_id"] == nil || int64(r.Num("reply_to_id")) != rootID {
		t.Fatalf("ответ потерял родителя: %s", r.Raw)
	}

	// Ответ на ответ — дерево растёт вглубь без ограничений.
	r = portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/posts/%d/comments", postID),
		mate.Token, map[string]any{"text": "И я тоже", "reply_to_id": replyID})
	requireStatus(t, r, 201, "ответ на ответ")
	deepID := int64(r.Num("id"))
	if int64(r.Num("reply_to_id")) != replyID {
		t.Fatalf("вложенный ответ потерял родителя: %s", r.Raw)
	}

	// Родитель обязан жить в этом же посте.
	r = portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/posts/%d/comments", otherPostID),
		mate.Token, map[string]any{"text": "чужая ветка", "reply_to_id": rootID})
	requireError(t, r, 404, "NOT_FOUND", "ответ на комментарий чужого поста")
	// Несуществующий родитель — тоже 404.
	r = portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/posts/%d/comments", postID),
		mate.Token, map[string]any{"text": "в пустоту", "reply_to_id": 999999})
	requireError(t, r, 404, "NOT_FOUND", "ответ на несуществующий комментарий")

	// Обсуждение приходит плоским списком с родителями — дерево строит клиент.
	r = portalAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/portal/posts/%d/comments", postID), author.Token, nil)
	requireStatus(t, r, 200, "список комментариев")
	list := r.List("comments")
	if len(list) != 3 {
		t.Fatalf("ожидалось 3 комментария, получено %d: %s", len(list), r.Raw)
	}
	if int64(findComment(t, list, deepID)["reply_to_id"].(float64)) != replyID {
		t.Fatalf("дерево не восстанавливается по reply_to_id: %s", r.Raw)
	}

	// Удаление корня уносит всю ветку.
	r = portalAPI.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/portal/comments/%d", rootID), mate.Token, nil)
	requireStatus(t, r, 200, "удаление корня ветки")
	r = portalAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/portal/posts/%d/comments", postID), author.Token, nil)
	if len(r.List("comments")) != 0 {
		t.Fatalf("ветка ответов должна уйти с родителем: %s", r.Raw)
	}
}

// TestPortalCommentLikes — лайк переключается одной ручкой, считается по
// людям (повторный лайк того же человека счётчик не наращивает) и виден
// каждому зрителю как «мой».
func TestPortalCommentLikes(t *testing.T) {
	admin := newVerifiedUser(t)
	companyID := admin.createCompany(t, uniq("Портал-лайки "))
	author := newMember(t, admin, companyID, roleEmployee)
	mate := newMember(t, admin, companyID, roleEmployee)
	stranger := newVerifiedUser(t)
	stranger.createCompany(t, uniq("Чужая ")) // свой скоуп — портал коллег ему не виден

	postID := createPost(t, author, "Пост с обсуждением")
	r := portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/posts/%d/comments", postID),
		mate.Token, map[string]any{"text": "Мысль"})
	requireStatus(t, r, 201, "комментарий")
	commentID := int64(r.Num("id"))

	// Свежий комментарий — без лайков.
	r = portalAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/portal/posts/%d/comments", postID), author.Token, nil)
	c := findComment(t, r.List("comments"), commentID)
	if c["like_count"] != float64(0) || c["liked"] != false {
		t.Fatalf("новый комментарий не может быть лайкнут: %s", r.Raw)
	}

	// Лайк автора поста.
	r = portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/comments/%d/like", commentID), author.Token, nil)
	requireStatus(t, r, 200, "лайк комментария")
	if !r.Bool("liked") || r.Num("like_count") != 1 {
		t.Fatalf("лайк не поставился: %s", r.Raw)
	}

	// Второй человек лайкает тот же комментарий — счётчик по людям.
	r = portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/comments/%d/like", commentID), mate.Token, nil)
	if !r.Bool("liked") || r.Num("like_count") != 2 {
		t.Fatalf("второй лайк: %s", r.Raw)
	}

	// «Мой лайк» — у каждого свой.
	r = portalAPI.doJSON(t, http.MethodGet, fmt.Sprintf("/api/portal/posts/%d/comments", postID), author.Token, nil)
	c = findComment(t, r.List("comments"), commentID)
	if c["like_count"] != float64(2) || c["liked"] != true {
		t.Fatalf("лайки в списке разъехались: %s", r.Raw)
	}

	// Повторный вызов той же ручкой — снятие лайка.
	r = portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/comments/%d/like", commentID), author.Token, nil)
	if r.Bool("liked") || r.Num("like_count") != 1 {
		t.Fatalf("повторный лайк должен сниматься: %s", r.Raw)
	}

	// Чужая компания комментария не видит и лайкнуть его не может.
	r = portalAPI.doJSON(t, http.MethodPost, fmt.Sprintf("/api/portal/comments/%d/like", commentID), stranger.Token, nil)
	requireError(t, r, 404, "NOT_FOUND", "лайк комментария чужой компании")
}
