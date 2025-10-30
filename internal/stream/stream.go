package stream

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

type FileInfo struct {
	From    string
	Where   []string
	Mode    fs.FileMode
	Size    int64
	IsDir   bool
	ModTime time.Time
}

type BufInfo struct {
	files   map[string]*FileInfo
	tomb    map[string]struct{}
	updTime time.Time
	mu      sync.RWMutex
}

// Запуск синхронизации заданой директории с буфером
func (buf *BufInfo) RunSync(path string, tmMult int, ctx context.Context) {
	var tm time.Time

	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(time.Millisecond *
				time.Duration(tmMult))
			err := buf.SyncFiles(path, &tm)
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	}
}

// Основная функция синхронизации директории и буфера
func (buf *BufInfo) SyncFiles(path string, modTime *time.Time) error {
	_, err := os.Stat(path)
	if err != nil {
		return err
	}

	check, _ := buf.comparePathInfo(path)
	if err != nil {
		slog.Warn("Can't get file names",
			"From path", path,
			"Error", err)
		return nil
	}

	if !check && buf.compareTime(modTime) {
		return nil
	}

	arr, err := MakePathArr(path)
	if err != nil {
		slog.Warn("Can't get file names",
			"From path", path,
			"Error", err)
		return nil
	}

	buf.updateFromPath(path, arr)

	bArr := buf.getAllNames()
	bArr = buf.FindDifer(arr, bArr)

	slices.Sort(*bArr)
	buf.updateFromBuf(path, bArr)

	buf.updateTime(modTime)

	return nil
}

// Синхронизация файлов из буфера и проверяемой директории
func (buf *BufInfo) updateFromBuf(path string, arr *[]string) {
	for _, name := range *arr {

		fInf := buf.TakeFileInfo(name)
		if fInf == nil {
			continue
		}

		if buf.inTomb(name) {
			if fInf.emptyInfo(path) {
				buf.delFromBuf(name)
			}
			continue
		}

		if fInf.findWhere(path) {
			buf.eraseWherePath(path, name)
			buf.addInTomb(name)
			slog.Info("Delete file from bufer",
				"From path", path,
				"File", name)
		} else {
			err := buf.BuildFile(path, name, fInf)
			if err != nil {
				continue
			}
			buf.addWherePath(path, name)
			slog.Info("Add file in dir",
				"To path", path,
				"File", name,
				"Size", (*fInf).Size)
		}

	}
	return
}

// Синхронизация файлов из проверяемой директории и буфера
func (buf *BufInfo) updateFromPath(path string, arr *[]string) {
	for _, name := range *arr {
		fullPath := filepath.Join(path, name)

		info, err := os.Stat(fullPath)
		if err != nil {
			if buf.findInfo(name) {
				buf.eraseWherePath(path, name)
				buf.addInTomb(name)
				slog.Info("Delete file from bufer",
					"From path", path,
					"File", name)
			}
			continue
		}

		fInf := buf.TakeFileInfo(name)
		if fInf == nil {
			buf.buildInfo(name, path, &info)
			slog.Info("Add in bufer",
				"From", path,
				"File", name,
				"Size", info.Size())
			continue
		}

		if !fInf.IsDir && !fInf.compareInfo(&info) {
			if buf.inTomb(name) {
				buf.delFromTomb(name)
			}
			if fInf.findWhere(path) {
				buf.buildInfo(name, path, &info)
				slog.Info("Add in bufer",
					"From", path,
					"File", name,
					"Size", (*fInf).Size)
			} else {
				err := os.RemoveAll(fullPath)
				if err != nil {
					slog.Error("Remove error",
						"Path", path,
						"File", name,
						"Error", err)
					continue
				}
				err = buf.BuildFile(path, name, fInf)
				if err != nil {
					slog.Error("Build error",
						"Path", path,
						"File", name,
						"Error", err)
					continue
				}
				buf.addWherePath(path, name)
				slog.Info("Rebuild file",
					"Path", path,
					"File", name,
					"Size", (*fInf).Size)
				continue
			}
		}

		if buf.inTomb(name) {
			buf.eraseWherePath(path, name)
			if fInf.emptyInfo(path) {
				buf.delFromBuf(name)
			}
			err := os.RemoveAll(fullPath)
			if err != nil {
				slog.Error("Remove error",
					"Path", path,
					"File", name,
					"Error", err)
				continue
			}
			slog.Info("File deleted",
				"From path", path,
				"File", name)
			continue
		}

		if !fInf.findWhere(path) {
			buf.addWherePath(path, name)
		}

	}
	return
}

// Получение всех имён файлов из буфера
func (buf *BufInfo) getAllNames() *[]string {
	(*buf).mu.RLock()
	defer (*buf).mu.RUnlock()
	var tmp = &[]string{}
	for name, _ := range (*buf).files {
		*tmp = append(*tmp, name)
	}
	return tmp
}

// Удаление информации из буфера
func (buf *BufInfo) delFromBuf(name string) {
	(*buf).mu.Lock()
	defer (*buf).mu.Unlock()
	delete((*buf).files, name)
	delete((*buf).tomb, name)
}

// Удаление имени файла из списка на удаление
func (buf *BufInfo) delFromTomb(name string) {
	(*buf).mu.Lock()
	defer (*buf).mu.Unlock()
	delete((*buf).tomb, name)
}

// Добавление имени файла в список на удаление
func (buf *BufInfo) addInTomb(name string) {
	(*buf).mu.Lock()
	defer (*buf).mu.Unlock()
	(*buf).tomb[name] = struct{}{}
	buf.setTime()
}

// Проверка наличия имени файла в списке на удаление и его наличие в буфере
func (buf *BufInfo) inTomb(name string) bool {
	(*buf).mu.RLock()
	defer (*buf).mu.RUnlock()
	if _, ok := (*buf).files[name]; !ok {
		return false
	}
	_, ok := (*buf).tomb[name]
	return ok
}

// Проверка актуальности информации в буфере
func (fInfo *FileInfo) emptyInfo(path string) bool {
	if len((*fInfo).Where) == 0 {
		return true
	}
	return false
}

// Поиск имени директории в списке информации о загрузке файла в буфере
func (fInfo *FileInfo) findWhere(path string) bool {
	for _, WherePath := range (*fInfo).Where {
		if WherePath == path {
			return true
		}
	}
	return false
}

// Удаляет из буфера имя директории, из которой был удалён файл
func (buf *BufInfo) eraseWherePath(path, name string) {
	(*buf).mu.Lock()
	defer (*buf).mu.Unlock()
	if _, ok := (*buf).files[name]; !ok {
		return
	}
	tmp := &((*buf).files[name].Where)
	for i, wherePath := range *tmp {
		if wherePath == path {
			*tmp = append((*tmp)[:i], (*tmp)[i+1:]...)
			break
		}
	}
}

// Добавляет в буфера имя директории, в которую записанн файл
func (buf *BufInfo) addWherePath(path, name string) {
	(*buf).mu.Lock()
	defer (*buf).mu.Unlock()
	if _, ok := (*buf).files[name]; !ok {
		return
	}
	(*buf).files[name].Where = append((*buf).files[name].Where, path)
}

// Поиск файлов в буфере, которых нет в проверяемой директории
func (buf *BufInfo) FindDifer(arr, bufArr *[]string) *[]string {
	var tmp = &[]string{}
	for _, nameBuf := range *bufArr {
		check := true
		for _, nameArr := range *arr {
			if nameBuf == nameArr {
				check = false
				break
			}
		}
		if check {
			*tmp = append(*tmp, nameBuf)
		}
	}
	return tmp
}

// Создание файла или директории по образу из буфера
func (buf *BufInfo) BuildFile(path, name string, fInfo *FileInfo) error {
	(*buf).mu.RLock()
	defer (*buf).mu.RUnlock()
	fullPath := filepath.Join(path, name)
	if (*fInfo).IsDir {
		err := os.MkdirAll(fullPath, (*fInfo).Mode)
		if err != nil {
			return err
		}
	} else {
		fromPath := (*fInfo).From
		data, err := os.ReadFile(fromPath)
		if err != nil {
			return err
		}

		err = os.WriteFile(fullPath, data, (*fInfo).Mode)
		if err != nil {
			return err
		}
		err = os.Chtimes(fullPath, (*fInfo).ModTime,
			(*fInfo).ModTime)
	}
	return nil
}

// Создание буфера с информацией о всех файлах в заданых директориях
func SyncInfo(paths []string) (*BufInfo, error) {
	buf := InitBufInfo()
	for _, path := range paths {

		arr, err := MakePathArr(path)
		if err != nil {
			return buf, err
		}

		buf.AddAllInfo(path, *arr)
		if err != nil {
			return buf, err
		}
	}
	slog.Info("All info is sync.", "Paths", paths)

	return buf, nil
}

// Добавление имён файлов и информацию о них в буфер
// или изменение уже существующей информации
func (buf *BufInfo) AddAllInfo(path string, names []string) {
	for _, name := range names {
		fullPath := filepath.Join(path, name)
		info, _ := os.Stat(fullPath)
		if !buf.findInfo(name) {
			buf.buildInfo(name, path, &info)
			continue
		}

		if (*buf).files[name].IsDir {
			buf.addWherePath(path, name)
			continue
		}

		if !buf.compareInfo(name, &info) && buf.isTimeAfter(name, &info) {
			buf.buildInfo(name, path, &info)
			continue
		}
	}
}

// Функция проверят, какой из файлов создан позже, имеющийся или предлагаемый
func (buf *BufInfo) isTimeAfter(name string, fInfo *os.FileInfo) bool {
	if (*buf).files[name].ModTime.After((*fInfo).ModTime()) {
		return true
	}
	return false
}

// Сравнение информации напрямую из буфера и проверяемого файла
func (buf *BufInfo) compareInfo(name string, info *os.FileInfo) bool {
	if !(*buf).files[name].ModTime.Equal((*info).ModTime()) {
		return false
	}
	if (*buf).files[name].Size != (*info).Size() {
		return false
	}
	if (*buf).files[name].Mode.Perm() != (*info).Mode().Perm() {
		return false
	}
	return true
}

// Сравнение информации взятой из буфера и проверяемого файла
func (fInfo *FileInfo) compareInfo(info *os.FileInfo) bool {
	if !(*fInfo).ModTime.Equal((*info).ModTime()) {
		return false
	}
	if (*fInfo).Size != (*info).Size() {
		return false
	}
	if (*fInfo).Mode.Perm() != (*info).Mode().Perm() {
		return false
	}
	return true
}

// Создание информации в буфере из файла
func (buf *BufInfo) buildInfo(name, path string, file *os.FileInfo) {
	(*buf).mu.Lock()
	defer (*buf).mu.Unlock()
	fullPath := filepath.Join(path, name)
	var tmp = []string{path}
	(*buf).files[name] = &FileInfo{
		From:    fullPath,
		Where:   tmp,
		Mode:    (*file).Mode(),
		Size:    (*file).Size(),
		ModTime: (*file).ModTime(),
		IsDir:   (*file).IsDir(),
	}
	buf.setTime()
}

// Создание массива имён файлов
func MakePathArr(path string) (*[]string, error) {
	var tmp = &[]string{}
	err := filepath.Walk(path, func(str string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if str == path {
			return nil
		}
		name, _ := filepath.Rel(path, str)

		*tmp = append(*tmp, name)

		return nil
	})
	if err != nil {
		return nil, err
	}
	return tmp, nil
}

// Создание массива имён файлов
func (buf *BufInfo) comparePathInfo(path string) (bool, error) {
	(*buf).mu.RLock()
	defer (*buf).mu.RUnlock()
	check := false
	pathLen := 0
	err := filepath.Walk(path, func(str string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if str == path {
			return nil
		}
		pathLen++
		name, _ := filepath.Rel(path, str)

		if _, ok := (*buf).files[name]; !ok {
			check = true
			return nil
		}

		if info.Size() != (*buf).files[name].Size {
			check = true
			return nil
		}
		if !info.IsDir() && !info.ModTime().Equal((*buf).files[name].ModTime) {
			check = true
			return err
		}
		if info.Mode() != (*buf).files[name].Mode {
			check = true
			return nil
		}

		return nil
	})
	if err != nil {
		return check, err
	}
	if pathLen != len((*buf).files) {
		check = true
	}
	return check, nil
}

// Сравнение времени изменения буфера и синхронизируемой директории
func (buf *BufInfo) compareTime(tm *time.Time) bool {
	(*buf).mu.RLock()
	defer (*buf).mu.RUnlock()
	return (*buf).updTime.Equal(*tm)
}

// Обновление времени изменения синхронизируемой директории
func (buf *BufInfo) updateTime(tm *time.Time) {
	(*buf).mu.RLock()
	defer (*buf).mu.RUnlock()
	*tm = (*buf).updTime
}

// Фиксирование времени изменения буфера
func (buf *BufInfo) setTime() {
	(*buf).updTime = time.Now()
}

// Поиск наличия информации о файле в буфере
func (buf *BufInfo) findInfo(name string) bool {
	(*buf).mu.RLock()
	defer (*buf).mu.RUnlock()
	if _, ok := (*buf).files[name]; !ok {
		return false
	}
	return true
}

// Инициализация структуры BufInfo
func InitBufInfo() *BufInfo {
	var buf = &BufInfo{
		files: make(map[string]*FileInfo),
		tomb:  make(map[string]struct{}),
	}
	return buf
}

// Взятие структуры FileInfo из структуры BufInfo по имени файла
func (buf *BufInfo) TakeFileInfo(name string) *FileInfo {
	(*buf).mu.RLock()
	defer (*buf).mu.RUnlock()
	if _, ok := (*buf).files[name]; !ok {
		return nil
	}
	tmp := (*(*buf).files[name])
	return &tmp
}

// Получение количества имён файлов из структуры BufInfo
func (buf *BufInfo) FilesLen() int {
	(*buf).mu.RLock()
	defer (*buf).mu.RUnlock()
	return len((*buf).files)
}
