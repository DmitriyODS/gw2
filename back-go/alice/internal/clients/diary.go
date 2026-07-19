package clients

import (
	"context"

	"google.golang.org/grpc"

	"github.com/DmitriyODS/gw2/back-go/alice/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/diarypb"
)

type Diary struct {
	conn *grpc.ClientConn
	stub diarypb.DiaryServiceClient
}

var _ domain.DiaryClient = (*Diary)(nil)

func NewDiary(addr string) (*Diary, error) {
	conn, err := dial(addr)
	if err != nil {
		return nil, err
	}
	return &Diary{conn: conn, stub: diarypb.NewDiaryServiceClient(conn)}, nil
}

func (c *Diary) Close() { _ = c.conn.Close() }

func pbDiary(d *diarypb.Diary) domain.Diary {
	return domain.Diary{
		ID: d.GetId(), Name: d.GetName(),
		ActiveCount: int(d.GetActiveCount()), DoneCount: int(d.GetDoneCount()),
	}
}

func pbEntry(e *diarypb.Entry) *domain.Entry {
	if e == nil {
		return nil
	}
	return &domain.Entry{
		ID: e.GetId(), DiaryID: e.GetDiaryId(), Date: e.GetDate(),
		Title: e.GetTitle(), Description: e.GetDescription(), Done: e.GetDone(),
	}
}

func (c *Diary) ListDiaries(ctx context.Context, userID int64) ([]domain.Diary, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	resp, err := c.stub.ListDiaries(ctx, &diarypb.ListDiariesRequest{UserId: userID})
	if err != nil {
		return nil, err
	}
	if resp.GetError() != nil {
		return nil, pbErr(resp.GetError())
	}
	out := make([]domain.Diary, 0, len(resp.GetDiaries()))
	for _, d := range resp.GetDiaries() {
		out = append(out, pbDiary(d))
	}
	return out, nil
}

func (c *Diary) CreateDiary(ctx context.Context, userID int64, name string) (*domain.Diary, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	resp, err := c.stub.CreateDiary(ctx, &diarypb.CreateDiaryRequest{UserId: userID, Name: name})
	if err != nil {
		return nil, err
	}
	if resp.GetError() != nil {
		return nil, pbErr(resp.GetError())
	}
	d := pbDiary(resp.GetDiary())
	return &d, nil
}

func (c *Diary) ListEntries(ctx context.Context, userID, diaryID int64, from, to string) ([]domain.Entry, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	resp, err := c.stub.ListEntries(ctx, &diarypb.ListEntriesRequest{
		UserId: userID, DiaryId: diaryID, From: from, To: to,
	})
	if err != nil {
		return nil, err
	}
	if resp.GetError() != nil {
		return nil, pbErr(resp.GetError())
	}
	out := make([]domain.Entry, 0, len(resp.GetEntries()))
	for _, e := range resp.GetEntries() {
		out = append(out, *pbEntry(e))
	}
	return out, nil
}

func (c *Diary) CreateEntry(ctx context.Context, userID, diaryID int64, date, title string) (*domain.Entry, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	resp, err := c.stub.CreateEntry(ctx, &diarypb.CreateEntryRequest{
		UserId: userID, DiaryId: diaryID, Date: date, Title: title,
	})
	if err != nil {
		return nil, err
	}
	if resp.GetError() != nil {
		return nil, pbErr(resp.GetError())
	}
	return pbEntry(resp.GetEntry()), nil
}

func (c *Diary) SetEntryDone(ctx context.Context, userID, diaryID, entryID int64, done bool) (*domain.Entry, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	resp, err := c.stub.SetEntryDone(ctx, &diarypb.SetEntryDoneRequest{
		UserId: userID, DiaryId: diaryID, EntryId: entryID, Done: done,
	})
	if err != nil {
		return nil, err
	}
	if resp.GetError() != nil {
		return nil, pbErr(resp.GetError())
	}
	return pbEntry(resp.GetEntry()), nil
}

func (c *Diary) MoveEntry(ctx context.Context, userID, diaryID, entryID int64, date string) (*domain.Entry, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	resp, err := c.stub.MoveEntry(ctx, &diarypb.MoveEntryRequest{
		UserId: userID, DiaryId: diaryID, EntryId: entryID, Date: date,
	})
	if err != nil {
		return nil, err
	}
	if resp.GetError() != nil {
		return nil, pbErr(resp.GetError())
	}
	return pbEntry(resp.GetEntry()), nil
}

func (c *Diary) DeleteEntry(ctx context.Context, userID, diaryID, entryID int64) error {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	resp, err := c.stub.DeleteEntry(ctx, &diarypb.DeleteEntryRequest{
		UserId: userID, DiaryId: diaryID, EntryId: entryID,
	})
	if err != nil {
		return err
	}
	return pbErr(resp.GetError())
}
