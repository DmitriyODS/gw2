package companydata

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DmitriyODS/gw2/back-go/pkg/gen/authpb"
)

// MaxMessageBytes — контент компании едет одним сообщением: у активной
// компании это тысячи задач с комментариями, и дефолтные 4 МБ gRPC она
// перерастает быстро. Тот же предел ставят себе и владельцы на приёме.
const MaxMessageBytes = 128 << 20

// Client — сторона ОРКЕСТРАТОРА (authsvc): пул подключений к владельцам.
type Client struct {
	conns map[string]*grpc.ClientConn
}

// Dial — подключения к владельцам по спецификации вида
// «tasks=tasks:9095,portal=portal:9102» (одна переменная окружения, как у
// владельцев файлов). Соединения ленивые, поэтому недоступный на старте
// сервис подняться не мешает.
func Dial(spec string) (*Client, error) {
	c := &Client{conns: map[string]*grpc.ClientConn{}}
	for _, pair := range strings.Split(spec, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		name, addr, ok := strings.Cut(pair, "=")
		name, addr = strings.TrimSpace(name), strings.TrimSpace(addr)
		if !ok || name == "" || addr == "" {
			continue
		}
		conn, err := grpc.NewClient(addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(MaxMessageBytes),
				grpc.MaxCallSendMsgSize(MaxMessageBytes),
			),
		)
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("company data %s: %w", name, err)
		}
		c.conns[name] = conn
	}
	return c, nil
}

func (c *Client) Close() {
	for _, conn := range c.conns {
		_ = conn.Close()
	}
}

// Sections — разделы, к которым есть подключение (в порядке вызова не важно:
// оркестратор ходит по своему списку).
func (c *Client) Sections() []string {
	out := make([]string, 0, len(c.conns))
	for name := range c.conns {
		out = append(out, name)
	}
	return out
}

// Has — настроен ли раздел. Ненастроенный молча пропускается: стенд поднимает
// не все сервисы, и архив собирается из того, что есть.
func (c *Client) Has(section string) bool {
	_, ok := c.conns[section]
	return ok
}

var errNoSection = errors.New("company data: раздел не подключён")

func (c *Client) Export(ctx context.Context, section string, companyID int64) (Export, error) {
	conn, ok := c.conns[section]
	if !ok {
		return Export{}, errNoSection
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	res, err := authpb.NewCompanyDataServiceClient(conn).ExportCompany(ctx, &authpb.ExportCompanyRequest{CompanyId: companyID})
	if err != nil {
		return Export{}, err
	}
	return Export{Payload: res.GetPayload(), FileKeys: res.GetFileKeys(), Count: int(res.GetCount())}, nil
}

func (c *Client) Import(ctx context.Context, section string, in Import) (int, error) {
	conn, ok := c.conns[section]
	if !ok {
		return 0, errNoSection
	}
	users := make(map[string]int64, len(in.Users))
	for orig, id := range in.Users {
		users[strconv.FormatInt(orig, 10)] = id
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	res, err := authpb.NewCompanyDataServiceClient(conn).ImportCompany(ctx, &authpb.ImportCompanyRequest{
		CompanyId: in.CompanyID,
		ActorId:   in.ActorID,
		Payload:   in.Payload,
		UserMap:   users,
		FileMap:   in.Files,
	})
	if err != nil {
		return 0, err
	}
	return int(res.GetCount()), nil
}

func parseID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}
