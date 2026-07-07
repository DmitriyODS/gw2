package grpc

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/status"

	"github.com/DmitriyODS/gw2/back-go/pkg/gen/portalpb"
	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/portal/internal/endpoint"
	gokitendpoint "github.com/go-kit/kit/endpoint"
)

// stubEndpoint — go-kit endpoint, возвращающий фиксированный (response, err),
// с захватом последнего request для проверки маппинга proto → endpoint.
func stubEndpoint(resp any, err error, captured *any) gokitendpoint.Endpoint {
	return func(_ context.Context, request any) (any, error) {
		if captured != nil {
			*captured = request
		}
		return resp, err
	}
}

func TestCreateSystemPost_MapsFields(t *testing.T) {
	var captured any
	eps := endpoint.Endpoints{
		CreateSystemPost: stubEndpoint(&domain.Post{ID: 77}, nil, &captured),
	}
	srv := NewServer(eps)

	resp, err := srv.CreateSystemPost(context.Background(), &portalpb.CreateSystemPostRequest{
		CompanyId: 42, AuthorUserId: 7, SystemKind: "pet_evolved",
		Title: "Эволюция!", Body: "Питомец вырос.",
	})
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if resp.GetError() != nil {
		t.Fatalf("unexpected business error: %+v", resp.GetError())
	}
	if resp.GetPostId() != 77 {
		t.Fatalf("post_id mismatch: %+v", resp)
	}
	req, ok := captured.(endpoint.CreateSystemPostReq)
	if !ok {
		t.Fatalf("captured request has wrong type: %T", captured)
	}
	if req.CompanyID != 42 || req.AuthorID != 7 || req.SystemKind != "pet_evolved" ||
		req.Title != "Эволюция!" || req.Body != "Питомец вырос." {
		t.Fatalf("request fields not forwarded correctly: %+v", req)
	}
}

func TestCreateSystemPost_BusinessErrorInBand(t *testing.T) {
	eps := endpoint.Endpoints{
		CreateSystemPost: stubEndpoint(nil, domain.ErrCompanyDisabled, nil),
	}
	srv := NewServer(eps)

	resp, err := srv.CreateSystemPost(context.Background(), &portalpb.CreateSystemPostRequest{CompanyId: 1})
	if err != nil {
		t.Fatalf("business errors must travel in-band, not as transport error: %v", err)
	}
	if resp.GetError() == nil || resp.GetError().Code != "COMPANY_DISABLED" || resp.GetError().HttpStatus != 403 {
		t.Fatalf("expected COMPANY_DISABLED/403 in Error field, got %+v", resp.GetError())
	}
}

func TestCreateSystemPost_TransportErrorBecomesInternal(t *testing.T) {
	eps := endpoint.Endpoints{
		CreateSystemPost: stubEndpoint(nil, errors.New("boom: db down"), nil),
	}
	srv := NewServer(eps)

	if _, err := srv.CreateSystemPost(context.Background(), &portalpb.CreateSystemPostRequest{CompanyId: 1}); err == nil {
		t.Fatalf("infrastructure error must surface as transport error")
	} else if st, ok := status.FromError(err); !ok || st.Code().String() != "Internal" {
		t.Fatalf("expected Internal, got %v", err)
	}
}
