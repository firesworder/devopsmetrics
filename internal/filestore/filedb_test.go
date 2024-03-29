package filestore

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/storage"
)

var metricCounter, _ = storage.NewMetric("PollCount", internal.CounterTypeName, int64(10))
var metricGauge, _ = storage.NewMetric("RandomValue", internal.GaugeTypeName, 12.133)

func TestNewFileStore(t *testing.T) {
	type args struct {
		storeFilePath string
	}
	tests := []struct {
		want *FileStore
		name string
		args args
	}{
		{
			name: "Test #1. Not empty filepath.",
			args: args{storeFilePath: "some/filepath.json"},
			want: &FileStore{StoreFilePath: "some/filepath.json"},
		},
		{
			name: "Test #1. Empty filepath",
			args: args{storeFilePath: ""},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewFileStore(tt.args.storeFilePath))
		})
	}
}

func TestFileStore_Read(t *testing.T) {
	tests := []struct {
		want          *storage.MemStorage
		name          string
		storeFilePath string
		wantErr       bool
	}{
		{
			name:          "Test #1. File db is not exist.",
			storeFilePath: "files_test/read_not_exist_test.json",
			want:          nil,
			wantErr:       true,
		},
		{
			name:          "Test #2. File db is empty.",
			storeFilePath: "files_test/read_empty_file_test.json",
			want:          nil,
			wantErr:       true,
		},
		{
			name:          "Test #3. File db content correct json memstorage object.",
			storeFilePath: "files_test/read_correct_ms_test.json",
			want: storage.NewMemStorage(map[string]storage.Metric{
				metricCounter.Name: *metricCounter,
				metricGauge.Name:   *metricGauge,
			}),
			wantErr: false,
		},
		{
			name:          "Test #4. File db content correct json, but not memstorage object.",
			storeFilePath: "files_test/not_ms.json",
			want:          nil,
			wantErr:       true,
		},
		{
			name:          "Test #5. File db content incorrect json.",
			storeFilePath: "files_test/incorrect_json.json",
			want:          nil,
			wantErr:       true,
		},
		{
			name:          "Test #6. Not existed filepath(no dirs and file).",
			storeFilePath: "tmp/devops-metrics-db.json",
			want:          nil,
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := FileStore{StoreFilePath: tt.storeFilePath}
			got, err := f.Read()
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFileStore_Write(t *testing.T) {
	tests := []struct {
		name          string
		ms            storage.MemStorage
		storeFilePath string
		wantContentAs string
		wantError     bool
	}{
		{
			name:          "Test #1. Empty storage. (new file)",
			ms:            *storage.NewMemStorage(map[string]storage.Metric{}),
			storeFilePath: "",
			wantContentAs: "files_test/read_empty_ms_test.json",
			wantError:     false,
		},
		{
			name: "Test #2. Filled storage. (new file)",
			ms: *storage.NewMemStorage(map[string]storage.Metric{
				metricCounter.Name: *metricCounter,
				metricGauge.Name:   *metricGauge,
			}),
			storeFilePath: "",
			wantContentAs: "files_test/read_correct_ms_test.json",
			wantError:     false,
		},
		{
			name: "Test #3. File with memstorage already exist.",
			ms: *storage.NewMemStorage(map[string]storage.Metric{
				metricCounter.Name: *metricCounter,
				metricGauge.Name:   *metricGauge,
			}),
			storeFilePath: "files_test/write_store_exist.json",
			wantContentAs: "files_test/read_correct_ms_test.json",
			wantError:     false,
		},
		{
			name: "Test #5. Not existed filepath.",
			ms: *storage.NewMemStorage(map[string]storage.Metric{
				metricCounter.Name: *metricCounter,
				metricGauge.Name:   *metricGauge,
			}),
			storeFilePath: "/tmp/some_path.json",
			wantContentAs: "files_test/read_correct_ms_test.json",
			wantError:     false,
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f FileStore
			if tt.storeFilePath != "" {
				f.StoreFilePath = tt.storeFilePath
			} else {
				f.StoreFilePath = fmt.Sprintf("files_test/write_test_%d.json", i)
				defer os.Remove(f.StoreFilePath)
			}
			err := f.Write(&tt.ms)
			assert.Equal(t, tt.wantError, err != nil)

			if !tt.wantError {
				AssertEqualFileContent(t, tt.wantContentAs, f.StoreFilePath)
			}
		})
	}
}
