package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/message"
)

func Test_updateMemStats(t *testing.T) {
	runtime.ReadMemStats(&memstats)
	allocMetricBefore := memstats.Alloc
	pollCountBefore := PollCount
	randomValueBefore := RandomValue

	// нагрузка, чтобы повлиять на значения параметров в runtime.memstats
	demoSlice := []string{"demo"}
	for i := 0; i < 100; i++ {
		demoSlice = append(demoSlice, "demo")
	}

	err := updateMemStats()
	require.NoError(t, err)
	allocMetricAfter := memstats.Alloc
	pollCountAfter := PollCount
	randomValueAfter := RandomValue

	assert.NotEqual(t, allocMetricBefore, allocMetricAfter, "metric values were not updated")
	assert.Equal(t, true, pollCountBefore+1 == pollCountAfter,
		"PollCount was not updated correctly")
	assert.NotEqual(t, randomValueBefore, randomValueAfter, "RandomValue was not updated")
}

func TestSendMetricByURL(t *testing.T) {
	type args struct {
		paramName  string
		paramValue interface{}
	}
	tests := []struct {
		name           string
		args           args
		wantRequestURL string
	}{
		{
			name:           "Test 1. Gauge metric.",
			args:           args{paramName: "Alloc", paramValue: gauge(12.133)},
			wantRequestURL: "/update/gauge/Alloc/12.133000",
		},
		{
			name:           "Test 2. Counter metric.",
			args:           args{paramName: "PollCount", paramValue: counter(10)},
			wantRequestURL: "/update/counter/PollCount/10",
		},
		{
			name:           "Test 3. Metric with unknown type.",
			args:           args{paramName: "Alloc", paramValue: int64(10)},
			wantRequestURL: "",
		},
		{
			name:           "Test 4. Metric with nil value.",
			args:           args{paramName: "Alloc", paramValue: nil},
			wantRequestURL: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actualRequestURL string
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				actualRequestURL = r.URL.Path
			}))
			defer svr.Close()
			ServerURL = svr.URL
			sendMetricByURL(tt.args.paramName, tt.args.paramValue)
			assert.Equal(t, tt.wantRequestURL, actualRequestURL)
		})
	}
}

func TestSendMetricByJson(t *testing.T) {
	int64Value, float64Value := int64(10), float64(12.133)

	type args struct {
		paramName  string
		paramValue interface{}
	}
	type wantRequest struct {
		contentType string
		msg         *message.Metrics
	}

	tests := []struct {
		name        string
		args        args
		envKey      string
		wantRequest *wantRequest
	}{
		{
			name:   "Test 1. Gauge metric.",
			args:   args{paramName: "RandomValue", paramValue: gauge(12.133)},
			envKey: "Ayayaka",
			wantRequest: &wantRequest{
				contentType: "application/json",
				msg: &message.Metrics{
					ID:    "RandomValue",
					MType: internal.GaugeTypeName,
					Value: &float64Value,
					Delta: nil,
					Hash:  "19742de723a08df1f3436d0b745ea7743c05520787cb32949497056fce1f7c70",
				},
			},
		},
		{
			name:   "Test 2. Counter metric.",
			args:   args{paramName: "PollCount", paramValue: counter(10)},
			envKey: "Ayayaka",
			wantRequest: &wantRequest{
				contentType: "application/json",
				msg: &message.Metrics{
					ID:    "PollCount",
					MType: internal.CounterTypeName,
					Value: nil,
					Delta: &int64Value,
					Hash:  "4ca29a927a89931245cd4ad0782383d0fe0df883d31437cc5b85dc4dad3247c4",
				},
			},
		},
		{
			name:        "Test 3. Metric with unknown type.",
			args:        args{paramName: "Alloc", paramValue: int32(10)},
			envKey:      "Ayayaka",
			wantRequest: nil,
		},
		{
			name:        "Test 4. Metric with nil value.",
			args:        args{paramName: "Alloc", paramValue: nil},
			envKey:      "Ayayaka",
			wantRequest: nil,
		},
		{
			name:   "Test 5. Gauge metric. Key(env) is not set",
			args:   args{paramName: "RandomValue", paramValue: gauge(12.133)},
			envKey: "",
			wantRequest: &wantRequest{
				contentType: "application/json",
				msg: &message.Metrics{
					ID:    "RandomValue",
					MType: internal.GaugeTypeName,
					Value: &float64Value,
					Delta: nil,
					Hash:  "",
				},
			},
		},
		{
			name:   "Test 6. Counter metric. Key(env) is not set",
			args:   args{paramName: "PollCount", paramValue: counter(10)},
			envKey: "",
			wantRequest: &wantRequest{
				contentType: "application/json",
				msg: &message.Metrics{
					ID:    "PollCount",
					MType: internal.CounterTypeName,
					Value: nil,
					Delta: &int64Value,
					Hash:  "",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotRequest *wantRequest
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotRequest = &wantRequest{}
				gotRequest.contentType = r.Header.Get("Content-Type")

				msg := message.Metrics{}
				err := json.NewDecoder(r.Body).Decode(&msg)
				require.NoError(t, err, "cannot decode request body")
				gotRequest.msg = &msg
			}))
			defer svr.Close()
			Env.Key = tt.envKey
			ServerURL = svr.URL
			sendMetricByJSON(tt.args.paramName, tt.args.paramValue)
			require.Equal(t, tt.wantRequest, gotRequest)
		})
	}
}

func TestSendMetrics(t *testing.T) {
	t.Skipf("test not actual for batch metric sending")

	metricsCount := 29
	var gotMetricsReq = make([]string, 0, metricsCount)
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMetricsReq = append(gotMetricsReq, r.URL.Path)
	}))
	defer svr.Close()
	ServerURL = svr.URL
	SendMetrics()
	assert.Lenf(t, gotMetricsReq, metricsCount, "Expected %d requests, got %d", metricsCount, len(gotMetricsReq))
}

func TestParseEnvArgs(t *testing.T) {
	tests := []struct {
		name      string
		cmdStr    string
		envVars   map[string]string
		wantEnv   Environment
		wantPanic bool
	}{
		{
			name:    "Test correct 1. Empty cmd args and env vars.",
			cmdStr:  "file.exe",
			envVars: map[string]string{},
			wantEnv: Environment{
				ServerAddress: "localhost:8080", PollInterval: 2 * time.Second, ReportInterval: 10 * time.Second,
			},
			wantPanic: false,
		},
		{
			name:    "Test correct 2. Set cmd args and empty env vars.",
			cmdStr:  "file.exe --a=localhost:3030 -r=15s -p=3s",
			envVars: map[string]string{},
			wantEnv: Environment{
				ServerAddress: "localhost:3030", PollInterval: 3 * time.Second, ReportInterval: 15 * time.Second,
			},
			wantPanic: false,
		},
		{
			name:   "Test correct 3. Empty cmd args and set env vars.",
			cmdStr: "file.exe",
			envVars: map[string]string{
				"ADDRESS": "localhost:3030", "REPORT_INTERVAL": "20s", "POLL_INTERVAL": "5s",
			},
			wantEnv: Environment{
				ServerAddress: "localhost:3030", PollInterval: 5 * time.Second, ReportInterval: 20 * time.Second,
			},
			wantPanic: false,
		},
		{
			name:   "Test correct 4. Set cmd args and set env vars.",
			cmdStr: "file.exe --a=cmd.site -r=15s -p=3s",
			envVars: map[string]string{
				"ADDRESS": "env.site", "REPORT_INTERVAL": "20s", "POLL_INTERVAL": "5s",
			},
			wantEnv: Environment{
				ServerAddress: "env.site", PollInterval: 5 * time.Second, ReportInterval: 20 * time.Second,
			},
			wantPanic: false,
		},
		{
			name:   "Test correct 5. Partially set cmd args and set env vars. Field ADDRESS",
			cmdStr: "file.exe --r=15s --p=3s",
			envVars: map[string]string{
				"ADDRESS": "env.site", "REPORT_INTERVAL": "20s", "POLL_INTERVAL": "5s",
			},
			wantEnv: Environment{
				ServerAddress: "env.site", PollInterval: 5 * time.Second, ReportInterval: 20 * time.Second,
			},
			wantPanic: false,
		},
		{
			name:   "Test correct 6. Set cmd args and partially set env vars. Field ADDRESS",
			cmdStr: "file.exe --a=cmd.site --r=15s --p=3s",
			envVars: map[string]string{
				"REPORT_INTERVAL": "20s", "POLL_INTERVAL": "5s",
			},
			wantEnv: Environment{
				ServerAddress: "cmd.site", PollInterval: 5 * time.Second, ReportInterval: 20 * time.Second,
			},
			wantPanic: false,
		},
		{
			name:   "Test 7. Field key, cmd",
			cmdStr: "file.exe --a=cmd.site --r=15s --p=3s -k=ad123a",
			envVars: map[string]string{
				"REPORT_INTERVAL": "20s", "POLL_INTERVAL": "5s",
			},
			wantEnv: Environment{
				ServerAddress:  "cmd.site",
				PollInterval:   5 * time.Second,
				ReportInterval: 20 * time.Second,
				Key:            "ad123a",
			},
			wantPanic: false,
		},
		{
			name:   "Test 8. Field key, env",
			cmdStr: "file.exe --a=cmd.site --r=15s --p=3s",
			envVars: map[string]string{
				"REPORT_INTERVAL": "20s", "POLL_INTERVAL": "5s", "KEY": "ad123b",
			},
			wantEnv: Environment{
				ServerAddress:  "cmd.site",
				PollInterval:   5 * time.Second,
				ReportInterval: 20 * time.Second,
				Key:            "ad123b",
			},
			wantPanic: false,
		},
		{
			name:   "Test 9. Field key, not set",
			cmdStr: "file.exe --a=cmd.site --r=15s --p=3s",
			envVars: map[string]string{
				"REPORT_INTERVAL": "20s", "POLL_INTERVAL": "5s",
			},
			wantEnv: Environment{
				ServerAddress:  "cmd.site",
				PollInterval:   5 * time.Second,
				ReportInterval: 20 * time.Second,
				Key:            "",
			},
			wantPanic: false,
		},
		{
			name:   "Test 10. Field 'RateLimit', cmd",
			cmdStr: "file.exe --a=cmd.site -l=2",
			envVars: map[string]string{
				"REPORT_INTERVAL": "20s", "POLL_INTERVAL": "5s",
			},
			wantEnv: Environment{
				ServerAddress:  "cmd.site",
				PollInterval:   5 * time.Second,
				ReportInterval: 20 * time.Second,
				Key:            "",
				RateLimit:      2,
			},
			wantPanic: false,
		},
		{
			name:   "Test 11. Field 'RateLimit', env",
			cmdStr: "file.exe --a=cmd.site --r=15s --p=3s -l=1",
			envVars: map[string]string{
				"REPORT_INTERVAL": "20s", "POLL_INTERVAL": "5s", "RATE_LIMIT": "3",
			},
			wantEnv: Environment{
				ServerAddress:  "cmd.site",
				PollInterval:   5 * time.Second,
				ReportInterval: 20 * time.Second,
				Key:            "",
				RateLimit:      3,
			},
			wantPanic: false,
		},
		{
			name:   "Test 12. Field 'RateLimit', not set",
			cmdStr: "file.exe --a=cmd.site --r=15s --p=3s",
			envVars: map[string]string{
				"REPORT_INTERVAL": "20s", "POLL_INTERVAL": "5s",
			},
			wantEnv: Environment{
				ServerAddress:  "cmd.site",
				PollInterval:   5 * time.Second,
				ReportInterval: 20 * time.Second,
				Key:            "",
				RateLimit:      0,
			},
			wantPanic: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаю env в дефолтные значения(обнулять я его не могу, т.к. flag линки потеряются)
			Env.ServerAddress = "localhost:8080"
			Env.ReportInterval = 10 * time.Second
			Env.PollInterval = 2 * time.Second
			Env.Key = ""
			Env.RateLimit = 0

			// удаляю переменные окружения, если они были до этого установлены
			for _, key := range [5]string{"ADDRESS", "REPORT_INTERVAL", "POLL_INTERVAL", "KEY", "RATE_LIMIT"} {
				err := os.Unsetenv(key)
				require.NoError(t, err)
			}
			// устанавливаю переменные окружения использованные для теста
			for key, value := range tt.envVars {
				err := os.Setenv(key, value)
				require.NoError(t, err)
			}
			// устанавливаю os.Args как эмулятор вызванной команды
			os.Args = strings.Split(tt.cmdStr, " ")

			// сама проверка корректности парсинга\получения ошибок
			if tt.wantPanic {
				assert.Panics(t, ParseEnvArgs)
			} else {
				ParseEnvArgs()
				assert.Equal(t, tt.wantEnv, Env)
			}
		})
	}
}

func Test_sendMetricsBatchByJSON(t *testing.T) {
	int64Value, float64Value := int64(10), float64(2.27)
	envKey := "Ayaka"
	type request struct {
		contentType string
		msgBatch    []message.Metrics
	}

	wantRequest := request{
		contentType: "application/json",
		msgBatch: []message.Metrics{
			{
				ID:    "PollCount",
				MType: internal.CounterTypeName,
				Value: nil,
				Delta: &int64Value,
				Hash:  "566384d8026a5429fcc20ccac3248f014da91cb8fbfe8cd47883088c1741b0eb",
			},
			{
				ID:    "RandomValue",
				MType: internal.GaugeTypeName,
				Value: &float64Value,
				Delta: nil,
				Hash:  "ceb416f4ef87553a09a82f2909bbbaffd2eff26d1b7c4a29bb61ea38433876d2",
			},
		},
	}
	args := map[string]interface{}{
		"PollCount":   counter(10),
		"RandomValue": gauge(2.27),
	}

	var gotRequest request
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequest = request{}
		gotRequest.contentType = r.Header.Get("Content-Type")

		err := json.NewDecoder(r.Body).Decode(&gotRequest.msgBatch)
		require.NoError(t, err, "cannot decode request body")
	}))
	defer svr.Close()
	Env.Key = envKey
	ServerURL = svr.URL
	sendMetricsBatchByJSON(args)
	sort.Slice(gotRequest.msgBatch, func(i, j int) bool {
		return gotRequest.msgBatch[i].ID < gotRequest.msgBatch[j].ID
	})
	assert.Equal(t, wantRequest, gotRequest)
}

func Test_updateGoPsutilStats(t *testing.T) {
	var err error
	err = updateGoPsutilStats()
	require.NoError(t, err)
	// before
	tmB, fmB, cpuUB := goPsutilStats.TotalMemory, goPsutilStats.FreeMemory, goPsutilStats.CPUutilization

	// нагрузка, чтобы повлиять на значения параметров
	demoSlice := []string{"demo"}
	for i := 0; i < 100; i++ {
		demoSlice = append(demoSlice, "demo")
	}

	err = updateGoPsutilStats()
	require.NoError(t, err)
	// after
	tmA, fmA, cpuUA := goPsutilStats.TotalMemory, goPsutilStats.FreeMemory, goPsutilStats.CPUutilization
	assert.Equal(t, tmB, tmA, "total memory stat differs")
	assert.NotEqual(t, fmB, fmA, "free memory has not changed")
	assert.NotEqual(t, cpuUB, cpuUA, "cpu utils stats have not changed")
}

func TestUpdateMetrics(t *testing.T) {
	// взять текущие значения метрик
	testUMWG = &sync.WaitGroup{}
	testUMWG.Add(1)
	go UpdateMetrics()
	testUMWG.Wait()
	memstatsBefore, goPsutilStatsBefore := memstats, goPsutilStats

	// нагрузка, чтобы повлиять на значения параметров
	demoSlice := []string{"demo"}
	for i := 0; i < 100; i++ {
		demoSlice = append(demoSlice, "demo")
	}

	// получить значения метрик после создания нагрузки
	testUMWG.Add(1)
	go UpdateMetrics()
	testUMWG.Wait()
	memstatsNow, goPsutilStatsNow := memstats, goPsutilStats

	// сравнение значений
	assert.NotEqual(t, memstatsNow, memstatsBefore)
	assert.NotEqual(t, goPsutilStatsNow, goPsutilStatsBefore)
	testUMWG = nil
}

func TestInitWorkPool(t *testing.T) {
	Env.RateLimit = 15
	wp := WorkPool{}
	require.NotPanics(t, wp.Start)
	assert.NotEqual(t, wp.ch, nil)
	wp.Close()
}

// Не обрабатывает вариант когда отправлено было больше запросов(заданий) чем требовалось!
func TestCreateSendMetricsJob(t *testing.T) {
	// данные для теста
	gotRequestCountCh := make(chan bool)
	Env.RateLimit = 3
	wp := WorkPool{}
	wp.Start()
	gotRequestCount := 0
	wantRequestCount := 5

	// запуск тестового сервера
	serverMutex := sync.Mutex{}
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverMutex.Lock()
		gotRequestCount++
		if gotRequestCount == wantRequestCount {
			gotRequestCountCh <- true
		}
		serverMutex.Unlock()
	}))
	defer svr.Close()
	ServerURL = svr.URL

	timeoutTime := time.Second * 2
	ctxWT, cancelCtx := context.WithTimeout(context.Background(), timeoutTime)
	defer cancelCtx()
	for i := 0; i < wantRequestCount; i++ {
		go wp.CreateSendMetricsJob(ctxWT)
	}

	select {
	case <-ctxWT.Done():
		t.Errorf("timeout exceeded")
	case <-gotRequestCountCh:
		wp.Close()
		assert.Equal(t, wantRequestCount, gotRequestCount)
	}
}
