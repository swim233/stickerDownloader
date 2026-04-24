package utils

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"

	"github.com/swim233/StickerDownloader/logger"
	_ "golang.org/x/image/webp"
)

type FormatConverter struct {
}

// 将 WebP 转换为 PNG
func (f FormatConverter) WebpToPNG(src []byte) (dist []byte, err error) {
	reader := bytes.NewReader(src)
	img, _, err := image.Decode(reader)
	if err != nil {
		logger.Error("转换格式时出错 ： %s", err.Error())

		return nil, err
	}

	var buffer bytes.Buffer
	err = png.Encode(&buffer, img)
	if err != nil {
		logger.Error("转换格式时出错 ： %s", err.Error())

		return nil, err
	}

	return buffer.Bytes(), nil
}

// 将 WebP 转换为 JPEG
func (f FormatConverter) WebpToJPEG(src []byte, quality int) (dist []byte, err error) {
	reader := bytes.NewReader(src)
	img, _, err := image.Decode(reader)
	if err != nil {
		logger.Error("转换格式时出错 ： %s", err.Error())
		return nil, err
	}

	var buffer bytes.Buffer
	options := &jpeg.Options{Quality: quality} // 设置 JPEG 压缩质量
	err = jpeg.Encode(&buffer, img, options)
	if err != nil {
		logger.Error("转换格式时出错 ： %s", err.Error())
		return nil, err
	}

	return buffer.Bytes(), nil
}
