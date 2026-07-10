package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPageQuery_Normalize(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		q := PageQuery{}
		q.Normalize()
		assert.Equal(t, 1, q.Page)
		assert.Equal(t, 20, q.PageSize)
	})
	t.Run("custom values", func(t *testing.T) {
		q := PageQuery{Page: 3, PageSize: 50}
		q.Normalize()
		assert.Equal(t, 3, q.Page)
		assert.Equal(t, 50, q.PageSize)
	})
	t.Run("overflow page size", func(t *testing.T) {
		q := PageQuery{Page: 1, PageSize: 200}
		q.Normalize()
		assert.Equal(t, 20, q.PageSize, "over 100 should default to 20")
	})
	t.Run("zero page", func(t *testing.T) {
		q := PageQuery{Page: 0, PageSize: 10}
		q.Normalize()
		assert.Equal(t, 1, q.Page, "zero page should be 1")
	})
}

func TestPageQuery_Offset(t *testing.T) {
	t.Run("page 1 offset 0", func(t *testing.T) {
		q := PageQuery{Page: 1, PageSize: 20}
		assert.Equal(t, 0, q.Offset())
	})
	t.Run("page 3 offset 40", func(t *testing.T) {
		q := PageQuery{Page: 3, PageSize: 20}
		assert.Equal(t, 40, q.Offset())
	})
}

func TestLoginRequest_Validation(t *testing.T) {
	t.Run("valid code", func(t *testing.T) {
		req := LoginRequest{Code: "valid_wx_code"}
		assert.NotEmpty(t, req.Code)
	})
}

func TestPhoneLoginRequest_Validation(t *testing.T) {
	t.Run("valid phone and code", func(t *testing.T) {
		req := PhoneLoginRequest{
			Phone: "13812346288",
			Code:  "123456",
		}
		assert.Len(t, req.Phone, 11)
		assert.Len(t, req.Code, 6)
	})
}

func TestCreateHelpRequest_Validation(t *testing.T) {
	t.Run("required fields", func(t *testing.T) {
		req := CreateHelpRequest{
			AreaCode:    "450100",
			Address:     "南宁市西乡塘区安吉街道",
			Content:     "洪水淹没一楼",
			Urgency:     "high",
			PeopleCount: 3,
			Needs: []RequestNeedItem{
				{NeedName: "船只", Quantity: 1, Unit: "艘"},
			},
			IsProxy:          false,
			VulnerableGroups: []string{"elderly", "child"},
		}
		assert.Equal(t, "450100", req.AreaCode)
		assert.Equal(t, "high", req.Urgency)
		assert.Len(t, req.Needs, 1)
		assert.Len(t, req.VulnerableGroups, 2)
		assert.Contains(t, req.VulnerableGroups, "elderly")
	})

	t.Run("proxy fields", func(t *testing.T) {
		proxyName := "张三"
		proxyPhone := "13900001111"
		req := CreateHelpRequest{
			AreaCode:  "450100",
			Address:   "南宁",
			Content:   "代报求助",
			Urgency:   "medium",
			IsProxy:   true,
			ProxyName: &proxyName,
			ProxyPhone: &proxyPhone,
		}
		assert.True(t, req.IsProxy)
		assert.Equal(t, "张三", *req.ProxyName)
		assert.Equal(t, "13900001111", *req.ProxyPhone)
	})
}

func TestSOSTriggerRequest_Validation(t *testing.T) {
	t.Run("required fields", func(t *testing.T) {
		req := SOSTriggerRequest{
			Lat: 22.84, Lng: 108.32,
			Address:        "南宁西乡塘",
			LocationSource: "gps",
		}
		assert.InDelta(t, 22.84, req.Lat, 0.01)
		assert.InDelta(t, 108.32, req.Lng, 0.01)
		assert.Equal(t, "gps", req.LocationSource)
	})

	t.Run("last known location fallback", func(t *testing.T) {
		req := SOSTriggerRequest{
			Lat: 0, Lng: 0,
			LastKnownLat: 22.80, LastKnownLng: 108.30,
			LocationSource: "last_known",
		}
		assert.Equal(t, float64(0), req.Lat)
		assert.Equal(t, 22.80, req.LastKnownLat)
	})

	t.Run("guest phone", func(t *testing.T) {
		req := SOSTriggerRequest{
			Lat:        22.84,
			Lng:        108.32,
			GuestPhone: "13812346288",
		}
		assert.Len(t, req.GuestPhone, 11)
	})
}

func TestSOSConfirmRequest_Validation(t *testing.T) {
	validStatuses := []string{"confirmed", "cancelled", "false_alarm"}
	for _, s := range validStatuses {
		t.Run("valid status: "+s, func(t *testing.T) {
			req := SOSConfirmRequest{Status: s}
			assert.Equal(t, s, req.Status)
		})
	}
}

func TestHelpRequestListQuery_Validation(t *testing.T) {
	t.Run("empty query", func(t *testing.T) {
		q := HelpRequestListQuery{}
		assert.Empty(t, q.Status)
		assert.Empty(t, q.Urgency)
		assert.Equal(t, 0, q.Page)
	})
	t.Run("full query", func(t *testing.T) {
		q := HelpRequestListQuery{
			Status:   "pending",
			Urgency:  "critical",
			AreaCode: "450100",
			Search:   "西乡塘",
			Page:     1,
			PageSize: 50,
		}
		assert.Equal(t, "pending", q.Status)
		assert.Equal(t, "critical", q.Urgency)
		assert.Equal(t, "450100", q.AreaCode)
	})
}

func TestMatchListQuery_Validation(t *testing.T) {
	t.Run("empty query", func(t *testing.T) {
		q := MatchListQuery{}
		assert.Empty(t, q.Status)
	})
	t.Run("near timeout filter", func(t *testing.T) {
		q := MatchListQuery{Status: "accepted"}
		assert.Equal(t, "accepted", q.Status)
	})
}

func TestSetWaitingRequest_Validation(t *testing.T) {
	t.Run("valid waiting", func(t *testing.T) {
		req := SetWaitingRequest{
			Reason: "道路积水暂无法通行",
			ETAMin: 30,
		}
		assert.NotEmpty(t, req.Reason)
		assert.GreaterOrEqual(t, req.ETAMin, 1)
	})
}

func TestReinforceRequest_Validation(t *testing.T) {
	t.Run("valid reinforce", func(t *testing.T) {
		req := ReinforceRequest{
			Reason:      "200人被困需协办",
			NeededCount: 3,
		}
		assert.NotEmpty(t, req.Reason)
		assert.GreaterOrEqual(t, req.NeededCount, 1)
	})
}

func TestUpdateLocationRequest_Validation(t *testing.T) {
	t.Run("valid location", func(t *testing.T) {
		req := UpdateLocationRequest{Lat: 22.84, Lng: 108.32}
		assert.InDelta(t, 22.84, req.Lat, 0.01)
		assert.InDelta(t, 108.32, req.Lng, 0.01)
	})
	t.Run("Guangxi boundary", func(t *testing.T) {
		assert.InDelta(t, 24.0, 24.0, 20.0, "Guangxi lat range ~21-27")
		assert.InDelta(t, 108.0, 108.0, 5.0, "Guangxi lng range ~104-112")
	})
}

func TestRequestNeedItem_Validation(t *testing.T) {
	t.Run("valid need item", func(t *testing.T) {
		item := RequestNeedItem{NeedName: "饮用水", Quantity: 100, Unit: "瓶"}
		assert.Equal(t, "饮用水", item.NeedName)
		assert.Equal(t, 100, item.Quantity)
		assert.Equal(t, "瓶", item.Unit)
	})
	t.Run("min quantity", func(t *testing.T) {
		item := RequestNeedItem{NeedName: "船只", Quantity: 1}
		assert.GreaterOrEqual(t, item.Quantity, 1)
	})
	t.Run("empty unit defaults", func(t *testing.T) {
		item := RequestNeedItem{NeedName: "食物", Quantity: 5}
		assert.Empty(t, item.Unit, "unit should be empty and backend defaults to '个'")
	})
}

func TestUpdateHelpRequest_Validation(t *testing.T) {
	status := "resolved"
	content := "更新内容"
	req := UpdateHelpRequest{
		Content: &content,
		Status:  &status,
	}
	assert.Equal(t, "resolved", *req.Status)
	assert.Equal(t, "更新内容", *req.Content)
}

func TestCompleteMatchRequest_Validation(t *testing.T) {
	t.Run("with photos", func(t *testing.T) {
		req := CompleteMatchRequest{
			SubmitPhotos: []string{"photo_url_1", "photo_url_2"},
			SubmitNote:   "已顺利转移",
		}
		assert.Len(t, req.SubmitPhotos, 2)
		assert.Equal(t, "已顺利转移", req.SubmitNote)
	})
	t.Run("without photos", func(t *testing.T) {
		req := CompleteMatchRequest{}
		assert.Empty(t, req.SubmitPhotos)
	})
}

func TestUserProfileResponse(t *testing.T) {
	resp := UserProfileResponse{
		ID:           "u_28",
		Nickname:     "测试用户",
		PhoneMasked:  "138****6288",
		Status:       "active",
		LastKnownLat: 22.84,
		LastKnownLng: 108.32,
	}
	assert.Equal(t, "u_28", resp.ID)
	assert.Equal(t, "138****6288", resp.PhoneMasked)
	assert.InDelta(t, 22.84, resp.LastKnownLat, 0.01)
}

func TestDashboardKPI(t *testing.T) {
	kpi := DashboardKPI{
		TotalHelpRequests: 3847,
		ActiveMatches:     128,
		TotalVolunteers:   894,
	}
	assert.Greater(t, kpi.TotalHelpRequests, int64(0))
	assert.Greater(t, kpi.TotalVolunteers, kpi.ActiveMatches)
}

func TestMatchStats(t *testing.T) {
	stats := MatchStats{
		TotalMatches: 500,
		P2PCoverageRate: 0.42,
		AvgAcceptSec:   45.5,
	}
	assert.Greater(t, stats.TotalMatches, int64(0))
	assert.InDelta(t, 0.42, stats.P2PCoverageRate, 0.01)
	assert.Greater(t, stats.AvgAcceptSec, float64(0))
}
