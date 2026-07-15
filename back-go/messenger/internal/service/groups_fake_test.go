package service

import (
	"context"
	"sort"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
)

// In-memory реализация групповых методов Repository для юнит-тестов.

func (r *fakeRepo) memberMap(convID int64) map[int64]*domain.Member {
	m := r.members[convID]
	if m == nil {
		m = map[int64]*domain.Member{}
		r.members[convID] = m
	}
	return m
}

func (r *fakeRepo) CreateGroup(_ context.Context, title string, avatarPath *string,
	creatorID int64, memberIDs []int64) (*domain.Conversation, error) {
	r.nextConv++
	c := &domain.Conversation{
		ID: r.nextConv, IsGroup: true, Title: &title, AvatarPath: avatarPath,
		CreatedBy: &creatorID, CreatedAt: r.tick(),
	}
	r.convs[c.ID] = c
	mm := r.memberMap(c.ID)
	mm[creatorID] = &domain.Member{ConversationID: c.ID, UserID: creatorID,
		Role: domain.RoleOwner, JoinedAt: r.tick(),
		CanManageMembers: true, CanEditInfo: true, CanPinMessages: true}
	for _, uid := range memberIDs {
		if uid == 0 || uid == creatorID || mm[uid] != nil {
			continue
		}
		mm[uid] = &domain.Member{ConversationID: c.ID, UserID: uid,
			Role: domain.RoleMember, JoinedAt: r.tick(),
			CanManageMembers: true, CanEditInfo: true, CanPinMessages: true}
	}
	cp := *c
	return &cp, nil
}

func (r *fakeRepo) ListGroupConversations(_ context.Context, userID int64) ([]*domain.Conversation, error) {
	var out []*domain.Conversation
	for id, c := range r.convs {
		if !c.IsGroup {
			continue
		}
		mm := r.members[id]
		me := mm[userID]
		if me == nil || me.HiddenAt != nil {
			continue
		}
		cp := *c
		cp.MyRole = me.Role
		cp.MyMuted = me.Muted
		cp.MyPinnedAt = me.PinnedAt
		cp.MyLastReadID = me.LastReadMessageID
		cp.MemberCount = len(mm)
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *fakeRepo) RenameGroup(_ context.Context, convID int64, title string) error {
	r.convs[convID].Title = &title
	return nil
}

func (r *fakeRepo) SetGroupAvatar(_ context.Context, convID int64, avatarPath *string) error {
	r.convs[convID].AvatarPath = avatarPath
	return nil
}

func (r *fakeRepo) SetInviteCode(_ context.Context, convID int64, code *string) error {
	r.convs[convID].InviteCode = code
	return nil
}

func (r *fakeRepo) FindByInviteCode(_ context.Context, code string) (*domain.Conversation, error) {
	for _, c := range r.convs {
		if c.IsGroup && c.InviteCode != nil && *c.InviteCode == code {
			cp := *c
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *fakeRepo) AddMember(_ context.Context, convID, userID int64, role string) error {
	mm := r.memberMap(convID)
	if m := mm[userID]; m != nil {
		m.HiddenAt = nil
		return nil
	}
	mm[userID] = &domain.Member{ConversationID: convID, UserID: userID, Role: role,
		JoinedAt: r.tick(), CanManageMembers: true, CanEditInfo: true, CanPinMessages: true}
	return nil
}

func (r *fakeRepo) RemoveMember(_ context.Context, convID, userID int64) error {
	delete(r.memberMap(convID), userID)
	return nil
}

func (r *fakeRepo) GetMember(_ context.Context, convID, userID int64) (*domain.Member, error) {
	m := r.members[convID][userID]
	if m == nil {
		return nil, nil
	}
	cp := *m
	return &cp, nil
}

func (r *fakeRepo) ListMembers(_ context.Context, convID int64) ([]*domain.Member, error) {
	var out []*domain.Member
	for _, m := range r.memberMap(convID) {
		cp := *m
		if r.users != nil {
			cp.User = r.users.users[m.UserID]
		}
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].JoinedAt.Before(out[j].JoinedAt) })
	return out, nil
}

func (r *fakeRepo) ListMemberMutes(_ context.Context, convID int64) (map[int64]bool, error) {
	out := map[int64]bool{}
	for id, m := range r.memberMap(convID) {
		out[id] = m.Muted
	}
	return out, nil
}

func (r *fakeRepo) UpdateMemberRole(_ context.Context, convID, userID int64, role string) error {
	r.members[convID][userID].Role = role
	return nil
}

func (r *fakeRepo) UpdateMemberRights(_ context.Context, convID, userID int64, m, i, p bool) error {
	mem := r.members[convID][userID]
	mem.CanManageMembers, mem.CanEditInfo, mem.CanPinMessages = m, i, p
	return nil
}

func (r *fakeRepo) SetMemberMute(_ context.Context, convID, userID int64, muted bool) error {
	r.members[convID][userID].Muted = muted
	return nil
}

func (r *fakeRepo) SetMemberPin(_ context.Context, convID, userID int64, pinned bool) error {
	m := r.members[convID][userID]
	if pinned {
		t := r.tick()
		m.PinnedAt = &t
	} else {
		m.PinnedAt = nil
	}
	return nil
}

func (r *fakeRepo) HideConversationMember(_ context.Context, convID, userID int64, hidden bool) error {
	m := r.members[convID][userID]
	if hidden {
		t := r.tick()
		m.HiddenAt = &t
	} else {
		m.HiddenAt = nil
	}
	return nil
}

func (r *fakeRepo) SetMemberRead(_ context.Context, convID, userID, lastMessageID int64) (bool, error) {
	m := r.members[convID][userID]
	if m == nil {
		return false, nil
	}
	if m.LastReadMessageID != nil && *m.LastReadMessageID >= lastMessageID {
		return false, nil
	}
	m.LastReadMessageID = &lastMessageID
	t := r.tick()
	m.LastReadAt = &t
	return true, nil
}

func (r *fakeRepo) ReadersOf(_ context.Context, convID, messageID, authorID int64) ([]*domain.Member, error) {
	var out []*domain.Member
	for _, m := range r.memberMap(convID) {
		if m.UserID == authorID || m.LastReadMessageID == nil || *m.LastReadMessageID < messageID {
			continue
		}
		cp := *m
		if r.users != nil {
			cp.User = r.users.users[m.UserID]
		}
		out = append(out, &cp)
	}
	return out, nil
}

func (r *fakeRepo) CountGroupUnread(_ context.Context, convIDs []int64, userID int64) (map[int64]int, error) {
	out := map[int64]int{}
	for _, cid := range convIDs {
		m := r.members[cid][userID]
		if m == nil {
			continue
		}
		var watermark int64
		if m.LastReadMessageID != nil {
			watermark = *m.LastReadMessageID
		}
		n := 0
		for _, msg := range r.convMessages(cid) {
			if msg.ID > watermark && (msg.SenderID == nil || *msg.SenderID != userID) {
				n++
			}
		}
		if n > 0 {
			out[cid] = n
		}
	}
	return out, nil
}

func (r *fakeRepo) TotalGroupUnread(ctx context.Context, userID int64) (int, error) {
	total := 0
	for id := range r.convs {
		if !r.convs[id].IsGroup {
			continue
		}
		cnt, _ := r.CountGroupUnread(ctx, []int64{id}, userID)
		total += cnt[id]
	}
	return total, nil
}
