package handler

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"

	_ "golang.org/x/image/webp" // 导入 WebP 解码器
)

type formatConverter struct {
}

// 将 WebP 转换为 PNG
func (f formatConverter) convertWebPToPNG(webp []byte) ([]byte, error) {
	reader := bytes.NewReader(webp)
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	err = png.Encode(&buffer, img)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// 将 WebP 转换为 JPEG
func (f formatConverter) convertWebPToJPEG(webp []byte, quality int) ([]byte, error) {
	reader := bytes.NewReader(webp)
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	options := &jpeg.Options{Quality: quality} // 设置 JPEG 压缩质量
	err = jpeg.Encode(&buffer, img, options)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
