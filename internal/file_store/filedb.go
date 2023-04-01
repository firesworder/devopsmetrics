package file_store

import (
	"encoding/json"
	"github.com/firesworder/devopsmetrics/internal/storage"
	"io"
	"os"
	"path/filepath"
	"time"
)

// todo: сделать частью memstorage
// todo: добавить инициализации writer/reader в рамках этого пакета?
type FileStore struct {
	StoreFilePath string
	StoreInterval time.Duration // todo: удалить параметр
}

func NewFileStore(storeFilePath string) *FileStore {
	// FileStore имеет смысл только с НЕ пустым путем к файлу
	if storeFilePath != "" {
		return &FileStore{StoreFilePath: storeFilePath}
	}
	return nil
}

func (f *FileStore) Write(memStorage storage.MetricRepository) error {
	//todo: реализовать отдельно инициал. файла f.StoreFilePath (в конструкторе)
	if _, err := os.Stat(f.StoreFilePath); os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(f.StoreFilePath), 0644)
		if err != nil {
			return err
		}
	}
	file, err := os.OpenFile(f.StoreFilePath, os.O_WRONLY|os.O_CREATE, 0644)
	defer file.Close()
	if err != nil {
		return err
	}

	jsonMS, err := json.Marshal(&memStorage)
	if err != nil {
		return err
	}

	_, err = file.Write(jsonMS)
	if err != nil {
		return err
	}
	return nil
}

// todo: нужно ли создавать файл при открытии?
func (f *FileStore) Read() (*storage.MemStorage, error) {
	file, err := os.OpenFile(f.StoreFilePath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}

	jsonMS, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	memStorage := storage.NewMemStorage(map[string]storage.Metric{})
	err = json.Unmarshal(jsonMS, &memStorage)
	if err != nil {
		return nil, err
	}
	return memStorage, nil
}