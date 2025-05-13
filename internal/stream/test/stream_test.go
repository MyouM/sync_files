package stream

import (
	"testing"
	"github.com/stretchr/testify/require"
	"os"
	"time"
)

func TestSyncFiles(t *testing.T) {
        req := require.New(t)
	var (
		paths = []string{"sync1", "sync2"}
		tm1 time.Time
		tm2 time.Time
	)
        buf, err := SyncInfo(paths)
	req.Equal(err, nil)
	_ = os.Mkdir("sync1/someDir", 0777)
	_ = buf.SyncFiles(paths[0], &tm1)
	_ = buf.SyncFiles(paths[1], &tm2)
	req.DirExists("sync2/someDir")
	_ = os.RemoveAll("sync2/someDir")
	_ = buf.SyncFiles(paths[1], &tm2)
	_ = buf.SyncFiles(paths[0], &tm1)
	req.NoDirExists("sync1/someDir") 
	data, _ := os.ReadFile("sync1/text.txt")
	_ = os.WriteFile("sync1/new.txt", data, 0777)
	_ = buf.SyncFiles(paths[0], &tm1)
	_ = buf.SyncFiles(paths[1], &tm2)
	req.FileExists("sync2/new.txt")
	_ = os.RemoveAll("sync1/new.txt")
	_ = buf.SyncFiles(paths[0], &tm1)
	_ = buf.SyncFiles(paths[1], &tm2)

}

func BenchmarkSyncFilesWithDelet(b *testing.B) {
        var (
		paths = []string{"sync1", "sync2"}
		tm1 time.Time
		tm2 time.Time
	)
	buf, _ := SyncInfo(paths) 
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		data, _ := os.ReadFile("sync1/text.txt")
		_ = os.WriteFile("sync1/new.txt", data, 0777)
		_ = os.Mkdir("sync1/someDir", 0777)
		b.StartTimer()
		_ = buf.SyncFiles(paths[0], &tm1)
		_ = buf.SyncFiles(paths[1], &tm2)
		b.StopTimer()
		_ = os.RemoveAll("sync1/someDir")
		_ = os.RemoveAll("sync1/new.txt")
		b.StartTimer()
		_ = buf.SyncFiles(paths[0], &tm1)
		_ = buf.SyncFiles(paths[1], &tm2)
	}
}

func BenchmarkSyncFilesWithoutDelet(b *testing.B) {
        var (
		paths = []string{"bench1", "bench2"}
		tm1 time.Time
		tm2 time.Time
	)
        buf, _ := SyncInfo(paths)
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
		_ = buf.SyncFiles(paths[0], &tm1)
		_ = buf.SyncFiles(paths[1], &tm2)
		_ = buf.SyncFiles(paths[0], &tm1)
		_ = buf.SyncFiles(paths[1], &tm2)
        }
}



func TestFindDifer(t *testing.T) {
        req := require.New(t)
        var (
		paths = []string{"sync1", "sync2"}
		arr = &[]string{}
	)
        buf, err := SyncInfo(paths)
	req.Equal(err, nil)
	bArr := buf.getAllNames()
	bArr = buf.FindDifer(arr, bArr)
	req.Equal(len(*bArr), 2)
	check := false
	for _, name := range *bArr {
		if name == "text.txt" {
			check = true
		}
	}
	req.Equal(check, true)
}



func BenchmarkBuildFile(b *testing.B) {
        var paths = []string{"build1", "build2"}
        buf, _ := SyncInfo(paths)
	fInf1 := buf.TakeFileInfo("text.txt")
	fInf2 := buf.TakeFileInfo("dir1")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buf.BuildFile(paths[1], "text.txt", fInf1)
		_ = buf.BuildFile(paths[1], "dir1", fInf2)
	}
}

func BenchmarkSyncInfo(b *testing.B) {
        var paths = []string{"sync1", "sync2"}
	for i := 0; i < b.N; i++ {
		_, _ = SyncInfo(paths)
	}
}


func TestSyncInfo(t *testing.T) {
	req := require.New(t)
	var (
		b *BufInfo
		paths = []string{"sync1", "sync2"}	
	)
	buf, err := SyncInfo(paths)
	req.Equal(err, nil)
	req.Equal(buf.FilesLen(), 2)
	req.IsType(buf, b)
	bf := buf.TakeFileInfo("dir1")
	req.Equal(bf.IsDir, true)
}


func TestAddAllInfo(t *testing.T) {
	req := require.New(t)
	buf := InitBufInfo()
	path := "sync1"
	names := []string{"text.txt", "dir1"}
	buf.AddAllInfo(path, names)
	b1 := buf.TakeFileInfo("dir1")
	b2 := buf.TakeFileInfo("text.txt")
	req.Equal(b1.IsDir, true)
	req.Equal(b2.From, "sync1/text.txt")
}

func BenchmarkMakePathArr(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = MakePathArr("sync1")
	}
}

func TestMakePathArr(t *testing.T) {
        req := require.New(t)
        var path = "sync1"
	arr, err := MakePathArr(path)
	req.Equal(err, nil)
	check := false
	count := 1
	for _, name := range *arr {
		if name == "text.txt" {
			if count == 1 {
				check = true
				count++
			} else {
				check = false
			}
		}

	}
	req.Equal(check, true)
}

func TestComparePathInfo(t *testing.T) {
        req := require.New(t)
	paths := []string{"sync1", "sync2"}
        buf, _ := SyncInfo(paths)
	check, _ := buf.ComparePathInfo(paths[0])
	req.Equal(check, false)
	check, _ = buf.ComparePathInfo(paths[1])
	req.Equal(check, false)
}

func BenchmarkComparePathInfo(b *testing.B) {
        var paths = []string{"sync1", "sync2"}
        buf, _ := SyncInfo(paths)
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
		_, _ = buf.ComparePathInfo(paths[0])
	}
}


func BenchmarkGetAllNames(b *testing.B) {
        var paths = []string{"sync1", "sync2"}
        buf, _ := SyncInfo(paths)
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
		_ = buf.getAllNames()
	}
}

