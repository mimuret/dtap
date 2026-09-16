package stdout_test

import (
	"bufio"
	"os"
	"testing"

	"github.com/mimuret/dtap/v2/pkg/testtool"
)


var benchDM = testtool.CreateValidDnstapMessage()

// 現状: os.Stdout へ2回の個別 Write (syscall x2)
func BenchmarkWrite_Unbuffered_TwoWrites(b *testing.B) {
	b.ReportAllocs()
	f, _ := os.Open("/dev/null")
	defer f.Close()

	buf, err := benchDM.ConvertV1JSON()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Write(buf)
		f.Write([]byte("\n"))
	}
}

// 案1: buf に "\n" を append してから1回の Write (syscall x1)
func BenchmarkWrite_Unbuffered_OneWrite(b *testing.B) {
	b.ReportAllocs()
	f, _ := os.Open("/dev/null")
	defer f.Close()

	buf, err := benchDM.ConvertV1JSON()
	if err != nil {
		b.Fatal(err)
	}
	line := append(buf, '\n')

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Write(line)
	}
}

// 案2: bufio.Writer (メッセージごとに Flush)
func BenchmarkWrite_Buffered_FlushPerMsg(b *testing.B) {
	b.ReportAllocs()
	f, _ := os.Open("/dev/null")
	defer f.Close()
	w := bufio.NewWriterSize(f, 256*1024)

	buf, err := benchDM.ConvertV1JSON()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Write(buf)
		w.WriteByte('\n')
		w.Flush()
	}
}

// 案3: bufio.Writer (Flush なし / バッファが溜まったら自動 Flush)
func BenchmarkWrite_Buffered_NoFlush(b *testing.B) {
	b.ReportAllocs()
	f, _ := os.Open("/dev/null")
	defer f.Close()
	w := bufio.NewWriterSize(f, 256*1024)

	buf, err := benchDM.ConvertV1JSON()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Write(buf)
		w.WriteByte('\n')
	}
	w.Flush()
}
