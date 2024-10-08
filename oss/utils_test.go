package oss

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"

	. "gopkg.in/check.v1"
)

type OssUtilsSuite struct{}

var _ = Suite(&OssUtilsSuite{})

func (s *OssUtilsSuite) TestUtilsTime(c *C) {
	c.Assert(GetNowSec() > 1448597674, Equals, true)
	c.Assert(GetNowNanoSec() > 1448597674000000000, Equals, true)
	c.Assert(len(GetNowGMT()), Equals, len("Fri, 27 Nov 2015 04:14:34 GMT"))
}

func (s *OssUtilsSuite) TestUtilsSplitFile(c *C) {
	localFile := "../sample/BingWallpaper-2015-11-07.jpg"

	// Num
	parts, err := SplitFileByPartNum(localFile, 4)
	c.Assert(err, IsNil)
	c.Assert(len(parts), Equals, 4)
	testLogger.Println("parts 4:", parts)
	for i, part := range parts {
		c.Assert(part.Number, Equals, i+1)
		c.Assert(part.Offset, Equals, int64(i*120512))
		c.Assert(part.Size, Equals, int64(120512))
	}

	parts, err = SplitFileByPartNum(localFile, 5)
	c.Assert(err, IsNil)
	c.Assert(len(parts), Equals, 5)
	testLogger.Println("parts 5:", parts)
	for i, part := range parts {
		c.Assert(part.Number, Equals, i+1)
		c.Assert(part.Offset, Equals, int64(i*96409))
	}

	_, err = SplitFileByPartNum(localFile, 10001)
	c.Assert(err, NotNil)

	_, err = SplitFileByPartNum(localFile, 0)
	c.Assert(err, NotNil)

	_, err = SplitFileByPartNum(localFile, -1)
	c.Assert(err, NotNil)

	_, err = SplitFileByPartNum("notexist", 1024)
	c.Assert(err, NotNil)

	// Size
	parts, err = SplitFileByPartSize(localFile, 120512)
	c.Assert(err, IsNil)
	c.Assert(len(parts), Equals, 4)
	testLogger.Println("parts 4:", parts)
	for i, part := range parts {
		c.Assert(part.Number, Equals, i+1)
		c.Assert(part.Offset, Equals, int64(i*120512))
		c.Assert(part.Size, Equals, int64(120512))
	}

	parts, err = SplitFileByPartSize(localFile, 96409)
	c.Assert(err, IsNil)
	c.Assert(len(parts), Equals, 6)
	testLogger.Println("parts 6:", parts)
	for i, part := range parts {
		c.Assert(part.Number, Equals, i+1)
		c.Assert(part.Offset, Equals, int64(i*96409))
	}

	_, err = SplitFileByPartSize(localFile, 0)
	c.Assert(err, NotNil)

	_, err = SplitFileByPartSize(localFile, -1)
	c.Assert(err, NotNil)

	_, err = SplitFileByPartSize(localFile, 10)
	c.Assert(err, NotNil)

	_, err = SplitFileByPartSize("noexist", 120512)
	c.Assert(err, NotNil)
}

func (s *OssUtilsSuite) TestUtilsFileExt(c *C) {
	c.Assert(strings.Contains(TypeByExtension("test.txt"), "text/plain"), Equals, true)
	c.Assert(TypeByExtension("test.jpg"), Equals, "image/jpeg")
	c.Assert(TypeByExtension("test.pdf"), Equals, "application/pdf")
	c.Assert(TypeByExtension("test"), Equals, "")
	c.Assert(strings.Contains(TypeByExtension("/root/dir/test.txt"), "text/plain"), Equals, true)
	c.Assert(strings.Contains(TypeByExtension("root/dir/test.txt"), "text/plain"), Equals, true)
	c.Assert(strings.Contains(TypeByExtension("root\\dir\\test.txt"), "text/plain"), Equals, true)
	c.Assert(strings.Contains(TypeByExtension("D:\\work\\dir\\test.txt"), "text/plain"), Equals, true)
}

func (s *OssUtilsSuite) TestGetPartEnd(c *C) {
	end := GetPartEnd(3, 10, 3)
	c.Assert(end, Equals, int64(5))

	end = GetPartEnd(9, 10, 3)
	c.Assert(end, Equals, int64(9))

	end = GetPartEnd(7, 10, 3)
	c.Assert(end, Equals, int64(9))
}

func (s *OssUtilsSuite) TestParseRange(c *C) {
	// InvalidRange bytes==M-N
	_, err := ParseRange("bytes==M-N")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "InvalidRange bytes==M-N")

	// InvalidRange ranges=M-N
	_, err = ParseRange("ranges=M-N")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "InvalidRange ranges=M-N")

	// InvalidRange ranges=M-N
	_, err = ParseRange("bytes=M-N")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "InvalidRange bytes=M-N")

	// InvalidRange ranges=M-
	_, err = ParseRange("bytes=M-")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "InvalidRange bytes=M-")

	// InvalidRange ranges=-N
	_, err = ParseRange("bytes=-N")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "InvalidRange bytes=-N")

	// InvalidRange ranges=-0
	_, err = ParseRange("bytes=-0")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "InvalidRange bytes=-0")

	// InvalidRange bytes=1-2-3
	_, err = ParseRange("bytes=1-2-3")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "InvalidRange bytes=1-2-3")

	// InvalidRange bytes=1-N
	_, err = ParseRange("bytes=1-N")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "InvalidRange bytes=1-N")

	// Ranges=M-N
	ur, err := ParseRange("bytes=1024-4096")
	c.Assert(err, IsNil)
	c.Assert(ur.Start, Equals, (int64)(1024))
	c.Assert(ur.End, Equals, (int64)(4096))
	c.Assert(ur.HasStart, Equals, true)
	c.Assert(ur.HasEnd, Equals, true)

	// Ranges=M-N,X-Y
	ur, err = ParseRange("bytes=1024-4096,2048-4096")
	c.Assert(err, IsNil)
	c.Assert(ur.Start, Equals, (int64)(1024))
	c.Assert(ur.End, Equals, (int64)(4096))
	c.Assert(ur.HasStart, Equals, true)
	c.Assert(ur.HasEnd, Equals, true)

	// Ranges=M-
	ur, err = ParseRange("bytes=1024-")
	c.Assert(err, IsNil)
	c.Assert(ur.Start, Equals, (int64)(1024))
	c.Assert(ur.End, Equals, (int64)(0))
	c.Assert(ur.HasStart, Equals, true)
	c.Assert(ur.HasEnd, Equals, false)

	// Ranges=-N
	ur, err = ParseRange("bytes=-4096")
	c.Assert(err, IsNil)
	c.Assert(ur.Start, Equals, (int64)(0))
	c.Assert(ur.End, Equals, (int64)(4096))
	c.Assert(ur.HasStart, Equals, false)
	c.Assert(ur.HasEnd, Equals, true)
}

func (s *OssUtilsSuite) TestAdjustRange(c *C) {
	// Nil
	start, end := AdjustRange(nil, 8192)
	c.Assert(start, Equals, (int64)(0))
	c.Assert(end, Equals, (int64)(8192))

	// 1024-4096
	ur := &UnpackedRange{true, true, 1024, 4095}
	start, end = AdjustRange(ur, 8192)
	c.Assert(start, Equals, (int64)(1024))
	c.Assert(end, Equals, (int64)(4096))

	// 1024-
	ur = &UnpackedRange{true, false, 1024, 4096}
	start, end = AdjustRange(ur, 8192)
	c.Assert(start, Equals, (int64)(1024))
	c.Assert(end, Equals, (int64)(8192))

	// -4096
	ur = &UnpackedRange{false, true, 1024, 4096}
	start, end = AdjustRange(ur, 8192)
	c.Assert(start, Equals, (int64)(4096))
	c.Assert(end, Equals, (int64)(8192))

	// Invalid range 4096-1024
	ur = &UnpackedRange{true, true, 4096, 1024}
	start, end = AdjustRange(ur, 8192)
	c.Assert(start, Equals, (int64)(0))
	c.Assert(end, Equals, (int64)(8192))

	// Invalid range -1-
	ur = &UnpackedRange{true, false, -1, 0}
	start, end = AdjustRange(ur, 8192)
	c.Assert(start, Equals, (int64)(0))
	c.Assert(end, Equals, (int64)(8192))

	// Invalid range -9999
	ur = &UnpackedRange{false, true, 0, 9999}
	start, end = AdjustRange(ur, 8192)
	c.Assert(start, Equals, (int64)(0))
	c.Assert(end, Equals, (int64)(8192))
}

func (s *OssUtilsSuite) TestUtilCheckBucketName(c *C) {
	err := CheckBucketName("a")
	c.Assert(err, NotNil)

	err = CheckBucketName("a11111111111111111111111111111nbbbbbbbbbbbbbbbbbbbbbbbbbbbqqqqqqqqqqqqqqqqqqqq")
	c.Assert(err, NotNil)

	err = CheckBucketName("-abcd")
	c.Assert(err, NotNil)

	err = CheckBucketName("abcd-")
	c.Assert(err, NotNil)

	err = CheckBucketName("abcD")
	c.Assert(err, NotNil)

	err = CheckBucketName("abc 1")
	c.Assert(err, NotNil)

	err = CheckBucketName("abc&1")
	c.Assert(err, NotNil)

	err = CheckBucketName("abc-1")
	c.Assert(err, IsNil)

	err = CheckBucketName("1bc-1")
	c.Assert(err, IsNil)

	err = CheckBucketName("111-1")
	c.Assert(err, IsNil)

	err = CheckBucketName("abc123-def1")
	c.Assert(err, IsNil)
}

func (s *OssUtilsSuite) TestGetRangeString(c *C) {
	rangeInfo := UnpackedRange{}
	rangeInfo.HasStart = true
	rangeInfo.HasEnd = true
	rangeInfo.Start = 10
	rangeInfo.End = 20
	rangeStr := GetRangeString(rangeInfo)
	c.Assert(rangeStr, Equals, "10-20")

	rangeInfo.HasStart = true
	rangeInfo.HasEnd = false
	rangeInfo.Start = 10
	rangeInfo.End = 20
	rangeStr = GetRangeString(rangeInfo)
	c.Assert(rangeStr, Equals, "10-")

	rangeInfo.HasStart = false
	rangeInfo.HasEnd = true
	rangeInfo.Start = 10
	rangeInfo.End = 20
	rangeStr = GetRangeString(rangeInfo)
	c.Assert(rangeStr, Equals, "-20")

	rangeInfo.HasStart = false
	rangeInfo.HasEnd = false
	rangeInfo.Start = 10
	rangeInfo.End = 20
	rangeStr = GetRangeString(rangeInfo)
	c.Assert(rangeStr, Equals, "")
}

func (s *OssUtilsSuite) TestLimitReadCloser(c *C) {
	// test LimitReadCloser
	str := RandStr(1024)
	r := strings.NewReader(str)
	lrc := LimitReadCloser(r, 1024)

	rb := make([]byte, 1024)
	n, err := lrc.Read(rb)
	c.Assert(n, Equals, 1024)
	c.Assert(err, IsNil)

	// test DiscardReadCloser
	r = strings.NewReader(str)
	drc := DiscardReadCloser{
		RC:      io.NopCloser(r),
		Discard: 100,
	}
	n, err = drc.Read(rb)
	c.Assert(n, Equals, 1024-100)
	c.Assert(err, IsNil)
}

func (s *OssUtilsSuite) TestEscapeXml(c *C) {
	str := "\x00\x01\x02\x03\x04\x05\x06\a\b\t\n\v\f\r\x0e\x0f\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1A\x1B\x1C\x1D\x1E\x1F"
	str = EscapeXml(str)
	esc := string([]byte("&#x00;")) + string([]byte("&#x01;")) + string([]byte("&#x02;")) + string([]byte("&#x03;")) + string([]byte("&#x04;")) + string([]byte("&#x05;")) + string([]byte("&#x06;")) + string([]byte("&#x07;")) + string([]byte("&#x08;")) + string([]byte("&#x9;")) + string(string([]byte("&#xA;"))+string([]byte("&#x0B;"))+string([]byte("&#x0C;"))+string([]byte("&#xD;"))+string([]byte("&#x0E;"))+string([]byte("&#x0F;"))+string([]byte("&#x10;"))+string([]byte("&#x11;"))+string([]byte("&#x12;"))+string([]byte("&#x13;"))+string([]byte("&#x14;"))+string([]byte("&#x15;"))+string([]byte("&#x16;"))+string([]byte("&#x17;"))+string([]byte("&#x18;"))+string([]byte("&#x19;"))+string([]byte("&#x1A;"))+string([]byte("&#x1B;"))+string([]byte("&#x1C;"))+string([]byte("&#x1D;"))+string([]byte("&#x1E;"))+string([]byte("&#x1F;")))
	c.Assert(str, Equals, esc)

	str = "<>&\"'"
	str = EscapeXml(str)
	esc = string(escLT) + string(escGT) + string(escAmp) + string(escQuot) + string(escApos)
	c.Assert(str, Equals, esc)
}

func (s *OssUtilsSuite) TestCheckObjectNameEx(c *C) {
	err := CheckObjectNameEx("?", true)
	c.Assert("object name is invalid, can't start with '?'", Equals, err.Error())

	err = CheckObjectNameEx("?123", true)
	c.Assert("object name is invalid, can't start with '?'", Equals, err.Error())

	err = CheckObjectNameEx("?", false)
	c.Assert(err, IsNil)

	err = CheckObjectNameEx("?123", false)
	c.Assert(err, IsNil)
}

func (s *OssUtilsSuite) TestisVerifyObjectStrict(c *C) {
	// default
	config := getDefaultOssConfig()
	flag := isVerifyObjectStrict(config)
	c.Assert(true, Equals, flag)

	config = &Config{}
	flag = isVerifyObjectStrict(config)
	c.Assert(false, Equals, flag)
	c.Assert(false, Equals, config.VerifyObjectStrict)

	config = &Config{}
	config.VerifyObjectStrict = true
	flag = isVerifyObjectStrict(config)
	c.Assert(true, Equals, flag)
	c.Assert(true, Equals, config.VerifyObjectStrict)

	flag = isVerifyObjectStrict(nil)
	c.Assert(true, Equals, flag)

	config = getDefaultOssConfig()
	config.AuthVersion = AuthV2
	flag = isVerifyObjectStrict(config)
	c.Assert(false, Equals, flag)
	c.Assert(true, Equals, config.VerifyObjectStrict)

	config.AuthVersion = AuthV4
	flag = isVerifyObjectStrict(config)
	c.Assert(false, Equals, flag)
	c.Assert(true, Equals, config.VerifyObjectStrict)

	config.AuthVersion = ""
	flag = isVerifyObjectStrict(config)
	c.Assert(true, Equals, flag)
	c.Assert(true, Equals, config.VerifyObjectStrict)

	//
	config = getDefaultOssConfig()
	config.VerifyObjectStrict = false
	flag = isVerifyObjectStrict(config)
	c.Assert(false, Equals, flag)
	c.Assert(false, Equals, config.VerifyObjectStrict)
	c.Assert(AuthV1, Equals, config.AuthVersion)
}

type seekerReaderStub struct {
	r    io.ReadSeeker
	bErr bool
	cErr bool
	eErr bool
}

func (r *seekerReaderStub) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

func (r *seekerReaderStub) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		if r.bErr {
			return 0, errors.New("SeekStart error")
		}
	case io.SeekCurrent:
		if r.cErr {
			return 0, errors.New("SeekCurrent error")
		}
	case io.SeekEnd:
		if r.eErr {
			return 0, errors.New("SeekEnd error")
		}
	}
	return r.r.Seek(offset, whence)
}

func (s *OssUtilsSuite) TestGetReaderLen(c *C) {
	data := "hello world"

	// bytes.Buffer
	b := bytes.NewBuffer([]byte(data))
	n, err := GetReaderLen(b)
	c.Assert(err, IsNil)
	c.Assert(int64(len(data)), Equals, n)

	// bytes.Reader
	br := bytes.NewReader([]byte(data))
	n, err = GetReaderLen(br)
	c.Assert(err, IsNil)
	c.Assert(int64(len(data)), Equals, n)

	// strings.Reader
	sr := strings.NewReader(data)
	n, err = GetReaderLen(sr)
	c.Assert(err, IsNil)
	c.Assert(int64(len(data)), Equals, n)

	// os.File
	filePath := RandLowStr(10)
	CreateFile(filePath, data, c)
	f, err := os.Open(filePath)
	c.Assert(err, IsNil)
	n, err = GetReaderLen(f)
	c.Assert(err, IsNil)
	c.Assert(int64(len(data)), Equals, n)

	f.Seek(0, io.SeekEnd)
	n, err = GetReaderLen(f)
	c.Assert(err, IsNil)
	c.Assert(int64(0), Equals, n)

	f.Seek(2, io.SeekStart)
	n, err = GetReaderLen(f)
	c.Assert(err, IsNil)
	c.Assert(int64(len(data)-2), Equals, n)

	// io.LimitedReader
	b = bytes.NewBuffer([]byte(data))
	lr := io.LimitReader(b, 3)
	n, err = GetReaderLen(lr)
	c.Assert(err, IsNil)
	c.Assert(int64(3), Equals, n)

	// LimitedReadCloser
	b = bytes.NewBuffer([]byte(data))
	lrc := LimitReadCloser(b, 4)
	n, err = GetReaderLen(lrc)
	c.Assert(err, IsNil)
	c.Assert(int64(4), Equals, n)

	// err
	n, err = GetReaderLen(nil)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "can't get reader content length,unkown reader type")
	c.Assert(int64(0), Equals, n)

	f.Seek(0, io.SeekEnd)
	f.Seek(2, io.SeekCurrent)
	n, err = GetReaderLen(f)
	c.Assert(err.Error(), Equals, "can't get reader content length,unkown reader type")
	c.Assert(int64(0), Equals, n)

	// has not Len() , Seek(), N
	b = bytes.NewBuffer([]byte(data))
	bc := io.NopCloser(b)
	n, err = GetReaderLen(bc)
	c.Assert(err.Error(), Equals, "can't get reader content length,unkown reader type")
	c.Assert(int64(0), Equals, n)

	// Seek error
	sef := &seekerReaderStub{
		r: f,
	}
	sef.Seek(0, io.SeekStart)
	n, err = GetReaderLen(sef)
	c.Assert(err, IsNil)
	c.Assert(int64(len(data)), Equals, n)

	sef.bErr = true
	n, err = GetReaderLen(sef)
	c.Assert(err.Error(), Equals, "SeekStart error")
	c.Assert(int64(0), Equals, n)

	sef.bErr = false
	sef.cErr = true
	n, err = GetReaderLen(sef)
	c.Assert(err.Error(), Equals, "SeekCurrent error")
	c.Assert(int64(0), Equals, n)

	sef.cErr = false
	sef.eErr = true
	n, err = GetReaderLen(sef)
	c.Assert(err.Error(), Equals, "SeekEnd error")
	c.Assert(int64(0), Equals, n)
}
