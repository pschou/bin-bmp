// Copyright 2022 pschou (https://github.com/pschou)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path"
	"strconv"

	"github.com/dsnet/compress/bzip2"
	//"golang.org/x/crypto/salsa20"
)

var version string
var imgWidth *string
var verbose *bool

func main() {
	flag.Usage = func() {
		_, f := path.Split(os.Args[0])
		fmt.Fprintf(os.Stderr, "bmp-bin,  Version: %s (https://github.com/pschou/bmp-bin)\n"+
			"A utility to convert a bin to a bmp to look for patterns, alignment is done on every\n"+
			"  4th byte, so 4 bytes -> 1 pixel.\n"+
			"NOTE: Only the first 3 bytes in a quad are used for RGB display, the 4th is omitted.\n"+
			"  The bottom left is byte 0 and the top row are the bytes up to N, then filled with 0s'.\n\n"+
			"Usage: %s [options] input.bin output.bmp\n\n", version, f)
		flag.PrintDefaults()
	}

	log.SetFlags(0)
	log.SetPrefix("bmp-bin: ")
	decode := flag.Bool("d", false, "Decode, reverse the translation, bmp to bin")
	compress := flag.Bool("c", false, "Compression test")
	verbose = flag.Bool("v", false, "Verbose")
	imgWidth = flag.String("w", "auto", "Image width for output BMP\n"+
		"auto - gives a 3/4 proportion image\n"+
		"### - specify the exact width for uniformity")

	flag.Parse()

	if flag.NArg() != 2 {
		fmt.Println("Input and Output file needed")
		os.Exit(1)
	}

	fi, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}

	fo, err := os.Create(flag.Arg(1))

	if *decode {
		err = Decode(fo, fi, *compress)
	} else {
		err = Encode(fo, fi, *compress)
	}

	if err != nil && !errors.Is(err, io.EOF) {
		log.Fatalf("error: %v", err)
	}
}

func Decode(w *os.File, r *os.File, compress bool) error {
	var h header
	var err error
	binary.Read(r, binary.LittleEndian, &h)
	//fmt.Println("size", h.FileSize)
	stat, _ := r.Stat()
	//fmt.Println("size", stat.Size())
	if int64(h.FileSize) != stat.Size() {
		return fmt.Errorf("File size mismatch %d = %d", h.FileSize, stat.Size())
	}
	//mr := io.MultiReader(bytes.NewReader([]byte{0x42, 0x5a, 0x68}), r)
	if compress {
		mr := io.MultiReader(bytes.NewReader([]byte{0x42, 0x5a, 0x68}), &io.LimitedReader{R: r, N: int64(h.ImageSize - h.YPixelsPerMeter)})
		bz2r, err := bzip2.NewReader(mr, nil)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, bz2r)
		if err != nil {
			return err
		}
		bz2r.Close()
	} else {
		lr := io.LimitedReader{R: r, N: int64(h.ImageSize - h.YPixelsPerMeter)}
		_, err = io.Copy(w, &lr)
		if err != nil {
			return err
		}
	}
	return nil
}

func Encode(w *os.File, r io.Reader, compress bool) error {
	h := &header{
		SigBM:         [2]byte{'B', 'M'},
		FileSize:      14 + 40,
		PixOffset:     14 + 40,
		DibHeaderSize: 40,
		Width:         0,
		Height:        0,
		ColorPlane:    1,
		Bpp:           32,
	}
	binary.Write(w, binary.LittleEndian, h)
	//header_size, _ := w.Seek(0, io.SeekCurrent)
	//fmt.Println("header size:", header_size)

	//f, _ := os.Create("test")
	//mw := io.MultiWriter(rtw, f)
	var total int64

	if compress {
		rtw := &removeThree{w: w}
		bz2w, err := bzip2.NewWriter(rtw, nil)
		if err != nil {
			return err
		}

		io.Copy(bz2w, r)
		bz2w.Close()
		total = rtw.pos - 3 //51
	} else {
		total, _ = io.Copy(w, r)
	}

	ntotal := (total + 4) &^ 4
	if *imgWidth != "auto" {
		i, err := strconv.ParseUint(*imgWidth, 10, 32)
		if err != nil {
			return err
		}
		h.Width = uint32(i)
	}
	if h.Width == 0 {
		h.Width = (uint32(math.Ceil(math.Sqrt(float64(ntotal)/3))) + 4) &^ 4
	}
	h.Height = uint32(math.Ceil(float64(total) / 4 / float64(h.Width)))
	if *verbose {
		fmt.Println("bytes written:", total)
		fmt.Println("h:", h.Height)
		fmt.Println("w:", h.Width)
		fmt.Println("h*w:", h.Height*h.Width*4)

		fmt.Println("extra to write:", 4*int64(h.Height)*int64(h.Width)-total)
	}
	w.Write(make([]byte, 4*int64(h.Height)*int64(h.Width)-total))
	h.XPixelsPerMeter = uint32(4*int64(h.Height)*int64(h.Width) - total)
	h.YPixelsPerMeter = h.XPixelsPerMeter
	h.ImageSize = uint32(4 * int64(h.Height) * int64(h.Width))
	h.FileSize += h.ImageSize
	if *verbose {
		fmt.Println("filesize:", h.FileSize)
	}

	w.Seek(0, io.SeekStart)
	binary.Write(w, binary.LittleEndian, h)
	return nil
}

type removeThree struct {
	pos int64
	w   io.Writer
}

func (rt *removeThree) Write(p []byte) (n int, err error) {
	if rt.pos < 3 && len(p) > 0 {
		rt.pos++
		//fmt.Printf("byte %x\n", p[0])
		n, err = rt.Write(p[1:])
		n++
		return
	}
	n, err = rt.w.Write(p)
	rt.pos += int64(n)
	return
}

type header struct {
	SigBM           [2]byte
	FileSize        uint32
	Resverved       [2]uint16
	PixOffset       uint32
	DibHeaderSize   uint32
	Width           uint32
	Height          uint32
	ColorPlane      uint16
	Bpp             uint16
	Compression     uint32
	ImageSize       uint32
	XPixelsPerMeter uint32
	YPixelsPerMeter uint32
	ColorUse        uint32
	ColorImportant  uint32
}
