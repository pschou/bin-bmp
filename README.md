# BIN-BMP

A utility to convert any binary file into a BMP file, align on the 4th byte,
and look for patterns.   Useful for looking into new and uncharted binaries and
doing protocol inspection.


```bash
$ ./bin-bmp -h
bmp-bin,  Version: 0.1.20220527.1010 (https://github.com/pschou/bmp-bin)
A utility to convert a bin to a bmp to look for patterns, alignment is done on every 4th byte, so 4 bytes -> 1 pixel.
NOTE: Only the first 3 bytes in a quad are used for RGB display, the 4th is omitted.

Usage: bin-bmp [options] input.bin output.bmp

  -c    Compression test
  -d    Reverse the encoding, bmp to bin
```
