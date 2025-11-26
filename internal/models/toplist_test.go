package models

import (
	"testing"
	"time"
)

func TestToplistConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *ToplistConfig
		wantErr bool
		errType error
	}{
		{
			name: "valid config",
			config: &ToplistConfig{
				ID:         "test-1",
				UserID:     "user-123",
				Name:       "Test Toplist",
				Metric:     MetricChangePct,
				TimeWindow: Window5m,
				SortOrder:  SortOrderDesc,
				Enabled:    true,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			config: &ToplistConfig{
				UserID:     "user-123",
				Name:       "Test Toplist",
				Metric:     MetricChangePct,
				TimeWindow: Window5m,
				SortOrder:  SortOrderDesc,
			},
			wantErr: true,
			errType: ErrInvalidToplistID,
		},
		{
			name: "missing name",
			config: &ToplistConfig{
				ID:         "test-1",
				UserID:     "user-123",
				Metric:     MetricChangePct,
				TimeWindow: Window5m,
				SortOrder:  SortOrderDesc,
			},
			wantErr: true,
			errType: ErrInvalidToplistName,
		},
		{
			name: "invalid metric",
			config: &ToplistConfig{
				ID:         "test-1",
				UserID:     "user-123",
				Name:       "Test Toplist",
				Metric:     ToplistMetric("invalid"),
				TimeWindow: Window5m,
				SortOrder:  SortOrderDesc,
			},
			wantErr: true,
			errType: ErrInvalidToplistMetric,
		},
		{
			name: "invalid time window",
			config: &ToplistConfig{
				ID:         "test-1",
				UserID:     "user-123",
				Name:       "Test Toplist",
				Metric:     MetricChangePct,
				TimeWindow: ToplistTimeWindow("invalid"),
				SortOrder:  SortOrderDesc,
			},
			wantErr: true,
			errType: ErrInvalidToplistTimeWindow,
		},
		{
			name: "invalid sort order",
			config: &ToplistConfig{
				ID:         "test-1",
				UserID:     "user-123",
				Name:       "Test Toplist",
				Metric:     MetricChangePct,
				TimeWindow: Window5m,
				SortOrder:  ToplistSortOrder("invalid"),
			},
			wantErr: true,
			errType: ErrInvalidToplistSortOrder,
		},
		{
			name: "system toplist (no user_id)",
			config: &ToplistConfig{
				ID:         "system-gainers-1m",
				UserID:     "", // System toplist
				Name:       "Gainers 1m",
				Metric:     MetricChangePct,
				TimeWindow: Window1m,
				SortOrder:  SortOrderDesc,
				Enabled:    true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ToplistConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != tt.errType {
				t.Errorf("ToplistConfig.Validate() error = %v, wantErr %v", err, tt.errType)
			}
		})
	}
}

func TestToplistConfig_IsSystemToplist(t *testing.T) {
	tests := []struct {
		name   string
		config *ToplistConfig
		want   bool
	}{
		{
			name: "system toplist",
			config: &ToplistConfig{
				ID:     "system-1",
				UserID: "",
			},
			want: true,
		},
		{
			name: "user toplist",
			config: &ToplistConfig{
				ID:     "user-1",
				UserID: "user-123",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsSystemToplist(); got != tt.want {
				t.Errorf("ToplistConfig.IsSystemToplist() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToplistUpdate_Validate(t *testing.T) {
	tests := []struct {
		name    string
		update  *ToplistUpdate
		wantErr bool
		errType error
	}{
		{
			name: "valid system update",
			update: &ToplistUpdate{
				ToplistID:   "gainers_1m",
				ToplistType: "system",
				Rankings:    []ToplistRanking{},
				Timestamp:   time.Now(),
			},
			wantErr: false,
		},
		{
			name: "valid user update",
			update: &ToplistUpdate{
				ToplistID:   "user-123-custom-1",
				ToplistType: "user",
				Rankings:    []ToplistRanking{},
				Timestamp:   time.Now(),
			},
			wantErr: false,
		},
		{
			name: "missing toplist ID",
			update: &ToplistUpdate{
				ToplistType: "system",
				Rankings:    []ToplistRanking{},
				Timestamp:   time.Now(),
			},
			wantErr: true,
			errType: ErrInvalidToplistID,
		},
		{
			name: "invalid toplist type",
			update: &ToplistUpdate{
				ToplistID:   "test-1",
				ToplistType: "invalid",
				Rankings:    []ToplistRanking{},
				Timestamp:   time.Now(),
			},
			wantErr: true,
			errType: ErrInvalidToplistType,
		},
		{
			name: "missing timestamp",
			update: &ToplistUpdate{
				ToplistID:   "test-1",
				ToplistType: "system",
				Rankings:    []ToplistRanking{},
				Timestamp:   time.Time{},
			},
			wantErr: true,
			errType: ErrInvalidTimestamp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.update.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ToplistUpdate.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != tt.errType {
				t.Errorf("ToplistUpdate.Validate() error = %v, wantErr %v", err, tt.errType)
			}
		})
	}
}

func TestGetSystemToplistRedisKey(t *testing.T) {
	tests := []struct {
		name   string
		metric ToplistMetric
		window ToplistTimeWindow
		want   string
	}{
		{
			name:   "change_pct 1m",
			metric: MetricChangePct,
			window: Window1m,
			want:   "toplist:change_pct:1m",
		},
		{
			name:   "volume 1d",
			metric: MetricVolume,
			window: Window1d,
			want:   "toplist:volume:1d",
		},
		{
			name:   "rsi",
			metric: MetricRSI,
			window: Window15m,
			want:   "toplist:rsi:15m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSystemToplistRedisKey(tt.metric, tt.window); got != tt.want {
				t.Errorf("GetSystemToplistRedisKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetUserToplistRedisKey(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		toplistID string
		want      string
	}{
		{
			name:      "user toplist",
			userID:    "user-123",
			toplistID: "custom-1",
			want:      "toplist:user:user-123:custom-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetUserToplistRedisKey(tt.userID, tt.toplistID); got != tt.want {
				t.Errorf("GetUserToplistRedisKey() = %v, want %v", got, tt.want)
			}
		})
	}
}


func TestGetSystemToplistType(t *testing.T) {
	tests := []struct {
		name      string
		metric    ToplistMetric
		window    ToplistTimeWindow
		isGainer  bool
		want      SystemToplistType
	}{
		{
			name:     "gainers 1m",
			metric:   MetricChangePct,
			window:   Window1m,
			isGainer: true,
			want:     SystemToplistGainers1m,
		},
		{
			name:     "losers 5m",
			metric:   MetricChangePct,
			window:   Window5m,
			isGainer: false,
			want:     SystemToplistLosers5m,
		},
		{
			name:     "volume 1d",
			metric:   MetricVolume,
			window:   Window1d,
			isGainer: false,
			want:     SystemToplistVolume1d,
		},
		{
			name:     "rsi high",
			metric:   MetricRSI,
			window:   Window15m,
			isGainer: true,
			want:     SystemToplistRSIHigh,
		},
		{
			name:     "rsi low",
			metric:   MetricRSI,
			window:   Window15m,
			isGainer: false,
			want:     SystemToplistRSILow,
		},
		{
			name:     "relative volume",
			metric:   MetricRelativeVolume,
			window:   Window15m,
			isGainer: false,
			want:     SystemToplistRelVolume,
		},
		{
			name:     "vwap dist high",
			metric:   MetricVWAPDist,
			window:   Window15m,
			isGainer: true,
			want:     SystemToplistVWAPDistHigh,
		},
		{
			name:     "vwap dist low",
			metric:   MetricVWAPDist,
			window:   Window15m,
			isGainer: false,
			want:     SystemToplistVWAPDistLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSystemToplistType(tt.metric, tt.window, tt.isGainer); got != tt.want {
				t.Errorf("GetSystemToplistType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToplistConfig_ToJSON(t *testing.T) {
	config := &ToplistConfig{
		ID:         "test-1",
		UserID:     "user-123",
		Name:       "Test Toplist",
		Description: "Test description",
		Metric:     MetricChangePct,
		TimeWindow: Window5m,
		SortOrder:  SortOrderDesc,
		Enabled:    true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	jsonData, err := config.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("ToJSON() returned empty data")
	}

	// Test round-trip
	config2, err := ToplistConfigFromJSON(jsonData)
	if err != nil {
		t.Fatalf("ToplistConfigFromJSON() error = %v", err)
	}

	if config2.ID != config.ID {
		t.Errorf("ToplistConfigFromJSON() ID = %v, want %v", config2.ID, config.ID)
	}
	if config2.Name != config.Name {
		t.Errorf("ToplistConfigFromJSON() Name = %v, want %v", config2.Name, config.Name)
	}
	if config2.Metric != config.Metric {
		t.Errorf("ToplistConfigFromJSON() Metric = %v, want %v", config2.Metric, config.Metric)
	}
}

