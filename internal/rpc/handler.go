package rpc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Auto-CQUPT-Plan/rollcall-go/internal/config"
	"github.com/Auto-CQUPT-Plan/rollcall-go/internal/edge"
	"github.com/Auto-CQUPT-Plan/rollcall-go/internal/lms"
	edgekitex "github.com/Auto-CQUPT-Plan/rollcall-go/internal/rpc/kitex_gen/edge"
)

// EdgeServiceImpl implements the EdgeService interface defined in the IDL.
type EdgeServiceImpl struct {
	lmsClient *lms.Client
	wsClient  *edge.WSClient
	poller    *edge.Poller
	log       *slog.Logger
}

func NewEdgeServiceImpl(lmsClient *lms.Client, wsClient *edge.WSClient, poller *edge.Poller) *EdgeServiceImpl {
	return &EdgeServiceImpl{
		lmsClient: lmsClient,
		wsClient:  wsClient,
		poller:    poller,
		log:       slog.With("component", "edge_rpc"),
	}
}

func (s *EdgeServiceImpl) Health(ctx context.Context) (resp *edgekitex.HealthResponse, err error) {
	return &edgekitex.HealthResponse{
		Status:   "ok",
		ClientId: config.ClientID,
	}, nil
}

func (s *EdgeServiceImpl) GetRollcalls(ctx context.Context) (resp []*edgekitex.Rollcall, err error) {
	rollcalls, err := s.lmsClient.GetRollcalls(ctx)
	if err != nil {
		return nil, err
	}

	resp = make([]*edgekitex.Rollcall, 0, len(rollcalls))
	for _, r := range rollcalls {
		resp = append(resp, &edgekitex.Rollcall{
			RollcallId:   int64(r.RollcallID),
			Source:       r.Source,
			Status:       r.Status,
			CourseTitle:  r.CourseTitle,
			RollcallTime: r.RollcallTime,
		})
	}
	return resp, nil
}

func (s *EdgeServiceImpl) GetPauseState(ctx context.Context) (resp *edgekitex.PauseState, err error) {
	return &edgekitex.PauseState{
		Pause: config.PauseSharedRollcall.Load(),
	}, nil
}

func (s *EdgeServiceImpl) SetPause(ctx context.Context, req *edgekitex.SetPauseRequest) (resp *edgekitex.SetPauseResponse, err error) {
	config.PauseSharedRollcall.Store(req.Pause)
	return &edgekitex.SetPauseResponse{
		Message: "success",
		Pause:   req.Pause,
	}, nil
}

func (s *EdgeServiceImpl) QRCheckin(ctx context.Context, req *edgekitex.QRCheckinRequest) (resp *edgekitex.OperationResponse, err error) {
	qrData := edge.ExtractQRData(req.Data)
	if qrData == "" {
		return nil, fmt.Errorf("invalid or expired QR data")
	}

	result := s.lmsClient.DoCheckin(ctx, int(req.RollcallId), "qr", map[string]interface{}{
		"data": qrData,
	})

	if result.Success {
		go s.wsClient.SendToCenter(map[string]interface{}{
			"type":          "rollcall_success",
			"client_id":     config.ClientID,
			"rollcall_type": "qr",
			"rollcall_data": qrData,
			"timestamp":     time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		})
		s.poller.TriggerPoll()
		return &edgekitex.OperationResponse{Message: "success"}, nil
	}

	return nil, fmt.Errorf("%s", result.ErrorCode)
}

func (s *EdgeServiceImpl) NumberCheckin(ctx context.Context, req *edgekitex.NumberCheckinRequest) (resp *edgekitex.OperationResponse, err error) {
	result := s.lmsClient.DoCheckin(ctx, int(req.RollcallId), "number", map[string]interface{}{
		"numberCode": req.NumberCode,
	})

	if result.Success {
		rollcalls, _ := s.lmsClient.GetRollcalls(ctx)
		var courseTitle string
		var courseLocation interface{}
		for _, rc := range rollcalls {
			if rc.RollcallID == int(req.RollcallId) {
				courseTitle = rc.CourseTitle
				courseLocation = s.poller.GetCourseLocationForRollcall(rc)
				break
			}
		}

		go s.wsClient.SendToCenter(map[string]interface{}{
			"type":            "rollcall_success",
			"client_id":       config.ClientID,
			"rollcall_type":   "number",
			"course_title":    courseTitle,
			"course_location": courseLocation,
			"rollcall_id":     req.RollcallId,
			"rollcall_number": req.NumberCode,
			"timestamp":       time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		})
		s.poller.TriggerPoll()
		return &edgekitex.OperationResponse{Message: "success"}, nil
	}

	return nil, fmt.Errorf("%s", result.ErrorCode)
}

func (s *EdgeServiceImpl) LocationCheckin(ctx context.Context, req *edgekitex.LocationCheckinRequest) (resp *edgekitex.OperationResponse, err error) {
	result := s.lmsClient.DoCheckin(ctx, int(req.RollcallId), "radar", map[string]interface{}{
		"lat": req.Lat,
		"lon": req.Lon,
	})

	if result.Success {
		s.poller.TriggerPoll()
		return &edgekitex.OperationResponse{Message: "success"}, nil
	}

	return nil, fmt.Errorf("%s", result.ErrorCode)
}

func (s *EdgeServiceImpl) BatchQRCheckin(ctx context.Context, req *edgekitex.BatchQRCheckinRequest) (resp *edgekitex.BatchQRCheckinResponse, err error) {
	qrData := edge.ExtractQRData(req.Data)
	if qrData == "" {
		return nil, fmt.Errorf("invalid or expired QR data")
	}

	rollcalls, err := s.lmsClient.GetRollcalls(ctx)
	if err != nil {
		return nil, err
	}

	var results []*edgekitex.CheckinResult_
	for _, rc := range rollcalls {
		if rc.Source != "qr" || rc.Status != "absent" {
			continue
		}

		res := s.lmsClient.DoCheckin(ctx, rc.RollcallID, "qr", map[string]interface{}{
			"data": qrData,
		})

		cr := &edgekitex.CheckinResult_{
			RollcallId: int64(rc.RollcallID),
		}
		if res.Success {
			cr.Status = "success"
			go s.wsClient.SendToCenter(map[string]interface{}{
				"type":          "rollcall_success",
				"client_id":     config.ClientID,
				"rollcall_type": "qr",
				"rollcall_data": qrData,
				"timestamp":     time.Now().UTC().Format("2006-01-02T15:04:05Z"),
			})
			s.poller.TriggerPoll()
		} else {
			cr.Status = "failed"
			cr.Error = &res.ErrorCode
		}
		results = append(results, cr)
	}

	return &edgekitex.BatchQRCheckinResponse{Results: results}, nil
}
